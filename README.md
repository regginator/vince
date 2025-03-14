[link/repo]: https://github.com/regginator/vince
[link/releases]: https://github.com/regginator/vince/releases
[link/latest-release]: https://github.com/regginator/vince/releases/latest
[link/commits]: https://github.com/regginator/vince/commits

[badge/latest-release]: https://img.shields.io/github/v/release/regginator/vince?label=latest%20release
[badge/last-modified]: https://img.shields.io/github/last-commit/regginator/vince?label=last%20modifed
[badge/license]: https://img.shields.io/github/license/regginator/vince?label=license

# ViNCe

[![latest release][badge/latest-release]][link/releases] [![last modified][badge/last-modified]][link/commits] [![license][badge/license]](LICENSE)

A fast, dedicated VNC authentication bruteforcing tool, written in Go

> [!WARNING]
> Only use tools like ViNCe against networks and systems you have explicit permission to test on, I am not responsible for any malpractice caused or malicious activity performed using ViNCe.

## Install

#### **Prebuilt Binaries**

Head over to the [latest release][link/latest-release], then download & unpack the binary for your respective platform. Prebuilt binaries are automatically published per every release for various systems and architectures

#### **`go install`**

You can also install or update Vince with Go's built-in `go install`, granted you have a *recent* version of [Go](https://go.dev) installed, and `~/go/bin` is accessible from PATH:

```sh
go install -v github.com/regginator/vince@latest
```

## Usage

See full usage documentation below:

```
ViNCe v0.1.0
MIT License | Copyright (c) 2025 reggie@latte.to
https://github.com/regginator/vince

 INFO  Provide VNC server address and port with the `-a` flag (e.g. "192.168.0.134", "10.13.33.37:5901")
       See further usage below

USAGE: ./vince [OPTION]...
  -a string
    	Target VNC server [address:port], port defaults to 5900 unless specified
  -auth string
    	(Planned) Force use of a specific authentication type [vnc, tight] (default "vnc")
  -chars string
    	If mode is raw, the character set used for permutations (default "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
  -conn string
    	Use connection type [tcp, udp] (default "tcp")
  -delay float
    	Delay between connections per worker thread
  -m string
    	Mode of bruteforce [wordlist, raw] (default "wordlist")
  -no-probe
    	Don't perform an initial connection handshake probe
  -packet-debug
    	Enables packet dump logging for debug (meant for use with one thread)
  -proxies string
    	Path to list of SOCKS(4/5) proxies to use for workers. If not provided, no proxies are used. File must be a txt list of proxies in the format "scheme://[username:pass@]host[:port]"
  -range string
    	If mode is raw, min/max number range for password combination length. May be either a single number, or 2 numbers in the format "1-6" (default "1-6")
  -retries int
    	Number of retry attempts per password for failed connections. -1 means infinite retries (default -1)
  -start uint
    	Start at index n in password iteration
  -t int
    	Number of simultaneous worker threads. The target server may only be able to handle so many, or it may restrict 1 connection per IP, so proceed with caution (default 1)
  -w string
    	If mode is wordlist, path to the wordlist file to source from
```

## License

```
MIT License

Copyright (c) 2025 reggie@latte.to

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
