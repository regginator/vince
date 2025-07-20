package main

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/regginator/vince/rfb"
)

func doInitProbe() {
	pterm.Info.Println("Performing initial probe.. 🛸")

	client := &rfb.Client{
		DestAddr:            realTargetAddr,
		ConnType:            *ConnType,
		PacketDebug:         *PacketDebug,
		IsNoVnc:             *IsNoVnc,
		NoVncIsWss:          *NoVncIsWss,
		NoVncWebsockifyPath: *NoVncWebsockifyPath,
		NoVncUserAgent:      *NoVncUserAgent,
	}

	if proxyPool != nil {
		proxyAddr, err := proxyPool.Get()
		if err != nil {
			pterm.Error.Printf("failed to get proxy from pool: %s\n", err)
			os.Exit(1)
		}

		client.ProxyAddr = proxyAddr
	}

	if err := client.Connect(); err != nil {
		pterm.Error.Printf("failed to connect to server: %s\n", err)
		os.Exit(1)
	}

	// Handshake
	if err := client.DoHandshake(); err != nil {
		pterm.Error.Printf("failed to perform connection handshake: %s\n", err)
		os.Exit(1)
	}

	client.Kill()

	serverInfoList := []pterm.BulletListItem{
		{
			Level:       0,
			Text:        fmt.Sprintf("Server protocol ver: %s", client.ServerProtoVer),
			BulletStyle: pterm.NewStyle(pterm.FgCyan),
		},
		{
			Level:       0,
			Text:        fmt.Sprintf("Negotiated protocol ver: %s", client.ProtoVer),
			BulletStyle: pterm.NewStyle(pterm.FgCyan),
		},
		{
			Level:       0,
			Text:        "Auth types:",
			BulletStyle: pterm.NewStyle(pterm.FgCyan),
		},
	}

	for _, authType := range client.SecurityTypes {
		authTypeName := "Unknown"

		switch authType {
		case rfb.VncAuthInvalid:
			authTypeName = "Invalid"
		case rfb.VncAuthNone:
			authTypeName = "None"
		case rfb.VncAuthBasic:
			authTypeName = "VNC Authentication"
		// Non-standard
		case rfb.VncAuthTight:
			authTypeName = "Tight"
		case rfb.VncAuthUltra:
			authTypeName = "Ultra"
		case rfb.VncAuthTls:
			authTypeName = "TLS"
		case rfb.VncAuthVenCrypt:
			authTypeName = "VeNCrypt"
		case rfb.VncAuthGtkVncSasl:
			authTypeName = "GTK-VNC SASL"
		case rfb.VncAuthMd5Hash:
			authTypeName = "MD5 hash authentication"
		case rfb.VncAuthColinDeanXvp:
			authTypeName = "Colin Dean xvp"
		}

		serverInfoList = append(serverInfoList, pterm.BulletListItem{
			Level:       1,
			Text:        fmt.Sprintf("%s (%d)", authTypeName, authType),
			BulletStyle: pterm.NewStyle(pterm.FgCyan),
			Bullet:      ">",
		})
	}

	fmt.Println()
	err := pterm.DefaultBulletList.WithItems(serverInfoList).Render()
	_ = err

	/*
		if slices.Contains(client.SecurityTypes, rfb.VncAuthNone) {
			pterm.Success.Printf("🎉 Server has none-auth enabled, you should be able to connect w/out a password\n")
			os.Exit(0)
		}
	*/
}
