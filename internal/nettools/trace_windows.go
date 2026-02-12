//go:build windows

package nettools

import (
	"fmt"
	"net"
)

// HTTPTrace performs a traceroute to the given host
// Note: Traceroute is not supported on Windows due to lack of required syscalls
func HTTPTrace(host string, port int) (string, error) {
	// Resolve the host to show at least the destination IP
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve host: %v", err)
	}

	var ip net.IP
	for _, resolvedIP := range ips {
		if resolvedIP.To4() != nil {
			ip = resolvedIP.To4()
			break
		}
	}

	if ip == nil {
		return "", fmt.Errorf("no IPv4 address found for host")
	}

	return fmt.Sprintf("Traceroute to %s (%s)\nNote: UDP traceroute is not supported on Windows.\nThis feature requires Linux-specific syscalls (IP_RECVERR, Recvmsg).\nPlease use the Windows 'tracert' command or run SmartCalc on Linux/macOS.", host, ip.String()), nil
}
