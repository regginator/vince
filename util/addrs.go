package util

import (
	"fmt"
	"net"
)

// Lookup an addr while preserving any specified port and preventing lookups for
// hosts that are already IPs
func LookupAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr // Addr doesn't appear to be specifying port
	}

	if ip := net.ParseIP(host); ip != nil {
		// Host is already an IP, no need to lookup as host
		if port == "" {
			return host, nil
		}
		return net.JoinHostPort(host, port), nil
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	} else if len(ips) == 0 {
		return "", fmt.Errorf("host \"%s\" didn't return any host addresses", host)
	}

	// Just choose the first addr in the DNS response
	if port == "" {
		return ips[0], nil
	}
	return net.JoinHostPort(ips[0], port), nil
}

func AddrWithDefaultPort(addr string, defaultPort string) string {
	// If this returns an error, a port hasn't been specified, otherwise the addr is just invalid and
	// will be resolved as such during an actual lookup
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return net.JoinHostPort(addr, defaultPort)
	}

	return addr // Port is already specified
}
