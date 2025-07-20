package rfb

import (
	"bytes"
	"crypto/des"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/bits"
	"net"
	"net/http"
	"net/url"
	"time"

	_ "github.com/bdandy/go-socks4"
	"github.com/gorilla/websocket"
	"github.com/regginator/vince/util"
	"golang.org/x/net/proxy"
)

// Light implementation of a client for RFC 6143, with support for some non-standard auth types
// https://datatracker.ietf.org/doc/html/rfc6143

type Client struct {
	// Params for frontend

	DestAddr    string
	ConnType    string
	ProxyAddr   string // Passed to url.Parse, e.g. "socks5://127.0.0.1:1080". If empty, proxy is not used
	PacketDebug bool   // Enables 2-way logging of packet hex dumps for debugging

	IsNoVnc             bool
	NoVncIsWss          bool
	NoVncWebsockifyPath string
	NoVncUserAgent      string

	// Meant to be set internally, but accessible by frontend

	Conn           net.Conn
	ProtoVer       string // Negotiated protocol version
	ServerProtoVer string // Protocol version that the server initially reports in its banner, not necessarily the negotiated proto ver
	SecurityTypes  []VncAuth
	SecurityResult SecurityResult
}

// I should NOT have to do this
type websocketNetConn struct {
	*websocket.Conn
}

func (c *websocketNetConn) Read(b []byte) (int, error) {
	_, out, err := c.Conn.ReadMessage()
	n := copy(b, out)
	return n, err
}

func (c *websocketNetConn) Write(b []byte) (int, error) {
	err := c.Conn.WriteMessage(websocket.BinaryMessage, b)
	return len(b), err // We're larping about how many bytes we wrote
}

func (c *websocketNetConn) SetDeadline(t time.Time) error {
	// MEGA LARPING
	return nil
}

// Internal recv and send wrappers

func (client *Client) read(buf []byte) (int, error) {
	n, err := client.Conn.Read(buf)
	if client.PacketDebug && err == nil {
		dump := hex.Dump(buf[:n])
		log.Printf("[RECV] (%s -> %s)\n%s\n", client.Conn.RemoteAddr().String(), client.Conn.LocalAddr().String(), dump)
	}

	return n, err
}

func (client *Client) write(buf []byte) (int, error) {
	n, err := client.Conn.Write(buf)
	if client.PacketDebug && err == nil {
		dump := hex.Dump(buf[:n])
		log.Printf("[SEND] (%s -> %s)\n%s\n", client.Conn.LocalAddr().String(), client.Conn.RemoteAddr().String(), dump)
	}

	return n, err
}

// Attempts to init a connection based on the addr and conn type, also supports using a proxy
// connType must be "tcp" or "udp"
func (client *Client) Connect() error {
	if client.DestAddr == "" {
		return fmt.Errorf("client.DestAddr not specified")
	}

	connType := client.ConnType
	if connType == "" {
		connType = "tcp"
	}

	if client.IsNoVnc {
		scheme := "ws"
		if client.NoVncIsWss {
			scheme = "wss"
		}

		u := url.URL{Scheme: scheme, Host: client.DestAddr, Path: client.NoVncWebsockifyPath}
		h := http.Header{}

		if client.NoVncUserAgent != "" {
			h.Set("User-Agent", client.NoVncUserAgent)
		}

		wsDialer := &websocket.Dialer{
			HandshakeTimeout:  45 * time.Second,
			EnableCompression: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		// Also support proxies
		if client.ProxyAddr != "" {
			proxyUrl, err := url.Parse(client.ProxyAddr)
			if err != nil {
				return err
			}
			proxyDialer, err := proxy.FromURL(proxyUrl, proxy.Direct)
			if err != nil {
				return err
			}

			wsDialer.NetDial = proxyDialer.Dial
		}

		conn, _, err := wsDialer.Dial(u.String(), h)
		if err != nil {
			return err
		}

		client.Conn = &websocketNetConn{conn}
	} else if client.ProxyAddr != "" {
		// Use proxy (SOCKS) for connection
		proxyUrl, err := url.Parse(client.ProxyAddr)
		if err != nil {
			return err
		}

		dialer, err := proxy.FromURL(proxyUrl, proxy.Direct)
		if err != nil {
			return err
		}

		conn, err := dialer.Dial(connType, client.DestAddr)
		if err != nil {
			return err
		}

		client.Conn = conn
	} else {
		// Normal dial
		conn, err := net.Dial(connType, client.DestAddr)
		if err != nil {
			return err
		}

		client.Conn = conn
	}

	return nil
}

// Explicitly close the connection from our end
func (client *Client) Kill() {
	if client.Conn != nil {
		client.Conn.Close()
	}
}

func (client *Client) DoHandshake() error {
	{
		buf := make([]byte, 12)
		if n, err := client.read(buf); err != nil {
			return fmt.Errorf("InitServerBanner: %s", err)
		} else if n != 12 {
			return fmt.Errorf("InitServerBanner: expected exactly 12 bytes, got (%d)", n)
		} else if !bytes.Equal(buf[0:4], []byte("RFB ")) {
			return fmt.Errorf("InitServerBanner: invalid RFB banner header (\"%s\")", string(buf[0:4]))
		}

		client.ServerProtoVer = string(buf[4:11])
	}

	switch client.ServerProtoVer {
	case RfbProtoVer_3_3:
		client.ProtoVer = RfbProtoVer_3_3
	case RfbProtoVer_3_7:
		client.ProtoVer = RfbProtoVer_3_7
	case RfbProtoVer_3_8:
		client.ProtoVer = RfbProtoVer_3_8
	// Non-standard
	case RfbProtoVer_3_889:
		client.ProtoVer = RfbProtoVer_3_889
	default:
		// Anything else we don't recognize must be treated as 3.3
		client.ProtoVer = RfbProtoVer_3_3
	}

	// Send server the protocol we are going to use (or otherwise treat as)
	if _, err := client.write([]byte(fmt.Sprintf("RFB %s\n", client.ProtoVer))); err != nil {
		return fmt.Errorf("ProtocolVersionResponse: %s", err)
	}

	// Deal with auth type stuff

	if client.ProtoVer == RfbProtoVer_3_3 {
		// For 3.3, client doesn't get to negotiate anything. Server sends the security
		// type that we must use as a u32(why???) that can only be 0, 1, or 2

		buf := make([]byte, 512)
		if _, err := client.read(buf); err != nil {
			return fmt.Errorf("SecurityHandshakeOptions: %s", err)
		}

		reader := bytes.NewReader(buf)

		var secType uint32
		if err := binary.Read(reader, binary.BigEndian, &secType); err != nil {
			return fmt.Errorf("SecurityHandshakeOptions: packet too small (%s)", err)
		} else if secType > 255 {
			return fmt.Errorf("SecurityHandshakeOptions: security type is too large (expected <=255, got %d)", secType)
		}

		if secType == 0 {
			reasonMsg, err := util.ReadU32String(reader, binary.BigEndian)
			if err != nil {
				reasonMsg = "<failed to get error reason from server>"
			}

			return fmt.Errorf("SecurityHandshakeOptions: no security type returned: %s", reasonMsg)
		}

		client.SecurityTypes = []VncAuth{VncAuth(secType)}
	} else { // Basically treat as RFC describes for 3.8
		buf := make([]byte, 512)
		if _, err := client.read(buf); err != nil {
			return fmt.Errorf("SecurityHandshakeOptions: %s", err)
		}

		reader := bytes.NewReader(buf)

		var numSecTypes uint8
		if err := binary.Read(reader, binary.BigEndian, &numSecTypes); err != nil {
			return fmt.Errorf("SecurityHandshakeOptions: packet too small: %s", err)
		}

		/*
			If number-of-security-types is zero, then for some reason the
			connection failed (e.g., the server cannot support the desired
			protocol version).  This is followed by a string describing the
			reason (where a string is specified as a length followed by that many
			ASCII characters)
		*/
		if numSecTypes == 0 {
			reasonMsg, err := util.ReadU32String(reader, binary.BigEndian)
			if err != nil {
				reasonMsg = "<failed to get error reason from server>"
			}

			return fmt.Errorf("SecurityHandshakeOptions: no security types returned: %s", reasonMsg)
		}

		secTypesList := make([]VncAuth, numSecTypes)
		for i := 0; i < int(numSecTypes); i++ {
			if err := binary.Read(reader, binary.BigEndian, &secTypesList[i]); err != nil {
				return fmt.Errorf("SecurityHandshakeOptions: server reported (%d) security types, but packet is shorter than expected: %s", numSecTypes, err)
			}

			// For sake of debugging and clarity, VncAuthInvalid should be handled by frontend
		}

		client.SecurityTypes = secTypesList
	}

	// It's now on the frontend impl to decide what security type it will use, handshake is done

	return nil
}

func (client *Client) readSecurityResult() error {
	buf := make([]byte, 512)
	if _, err := client.read(buf); err != nil {
		return fmt.Errorf("SecurityResult: %s", err)
	}

	reader := bytes.NewReader(buf)

	// Just based on observation, for whatever reason Apple VNC server returns the status code
	// in little-endian for some reason, idk about anything else
	var order binary.ByteOrder = binary.BigEndian
	if client.ServerProtoVer == RfbProtoVer_3_889 {
		order = binary.LittleEndian
	}

	var statusCode uint32
	if err := binary.Read(reader, order, &statusCode); err != nil {
		return fmt.Errorf("SecurityResult: packet too small: %s", err)
	}

	switch statusCode {
	case 0: // OK
		client.SecurityResult.Success = true
	case 1: // failed
		reason, _ := util.ReadU32String(reader, binary.BigEndian)
		// If error, reason str is left empty

		client.SecurityResult.Reason = reason
	default:
		return fmt.Errorf("SecurityResult: invalid status code: expected [0, 1], got (%d)", statusCode)
	}

	return nil
}

// Auth submission impls for supported types
//
// Note that actual SecurityResult contents is put into client.SecurityResult, errors
// are still only returned for actual connection errs and such, not auth failed

func (client *Client) SubmitAuthNone() error {
	if client.ProtoVer == RfbProtoVer_3_3 {
		// For 3.3, there is no SecurityResult for None auth
		return nil
	}

	// Tell server that we are using auth type 1
	{
		payload := []byte{byte(VncAuthNone)}
		if _, err := client.write(payload); err != nil {
			return fmt.Errorf("SecurityHandshakeResponse: %s", err)
		}
	}

	if client.ProtoVer == RfbProtoVer_3_7 {
		// From here, 3.7 would just jump straight to VNC init messages
		return nil
	}

	if err := client.readSecurityResult(); err != nil {
		return fmt.Errorf("SecurityHandshakeResponse: %s", err)
	}

	return nil
}

func (client *Client) SubmitAuthBasic(password string) error {
	// Tell server that we are using auth type 1
	if client.ProtoVer != RfbProtoVer_3_3 {
		payload := []byte{byte(VncAuthBasic)}
		if _, err := client.write(payload); err != nil {
			return fmt.Errorf("SecurityHandshakeResponse: %s", err)
		}
	}

	// Auth challenge w/ DES
	{
		desChallengeBuf := make([]byte, 16)
		if n, err := client.read(desChallengeBuf); err != nil {
			return fmt.Errorf("BasicAuthChallenge: %s", err)
		} else if n < 16 {
			return fmt.Errorf("BasicAuthChallenge: packet too small")
		} else if bytes.Equal(desChallengeBuf[:16], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
			return fmt.Errorf("BasicAuthChallenge: server is likely a honeypot, DES challenge is all 0s")
		}

		if len(password) > 8 {
			// Password must be truncated to 8 chars
			password = password[:8]
		}

		pwBytes := make([]byte, 8)
		for i := 0; i < len(password); i++ {
			pwBytes[i] = bits.Reverse8(uint8(password[i]))
		}

		cipher, err := des.NewCipher(pwBytes)
		if err != nil {
			return fmt.Errorf("BasicAuthChallengeResponse: failed to create DES cipher from password \"%s\": %s", string(pwBytes), err)
		}

		desResult1 := make([]byte, 8)
		desResult2 := make([]byte, 8)
		cipher.Encrypt(desResult1, desChallengeBuf)
		cipher.Encrypt(desResult2, desChallengeBuf[8:])

		desResponseBuf := append(desResult1, desResult2...)

		if _, err := client.write(desResponseBuf); err != nil {
			return fmt.Errorf("BasicAuthChallengeResponse: %s", err)
		}
	}

	if err := client.readSecurityResult(); err != nil {
		return err
	}

	return nil
}
