package rfb

// Recognized protocol versions
const (
	RfbProtoVer_3_3 = "003.003"
	RfbProtoVer_3_7 = "003.007"
	RfbProtoVer_3_8 = "003.008"

	// Non-standard
	RfbProtoVer_3_889 = "003.889" // Apple remote desktop
)

// Recognized VNC authentication types
type VncAuth uint8

const (
	VncAuthInvalid VncAuth = iota
	VncAuthNone            // no auth
	VncAuthBasic           // aka "VNC Authentication"

	// Non-standard auth types
	// All of 3 to 15, as well as 128 to 255 are technicalllllly assigned to RealVNC, so we'll skip iota by 13

	VncAuthTight VncAuth = iota + 13
	VncAuthUltra
	VncAuthTls
	VncAuthVenCrypt
	VncAuthGtkVncSasl
	VncAuthMd5Hash
	VncAuthColinDeanXvp
)

type SecurityResult struct {
	Success bool
	Reason  string
}
