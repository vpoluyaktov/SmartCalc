//go:build linux || darwin

package nettools

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

// HopInfo represents information about a single hop
type HopInfo struct {
	Hop      int
	IP       string
	Hostname string
	RTTs     []time.Duration
	Timeout  bool
}

// HTTPTrace performs a UDP-based traceroute showing all hops along the path
// Uses UDP packets to high ports with incrementing TTL values (like standard traceroute)
//
// This implementation does NOT require root privileges because it uses UDP sockets
// instead of raw ICMP sockets. It listens for ICMP Time Exceeded messages that are
// sent back by intermediate routers.
func HTTPTrace(host string, port int) (string, error) {
	// Resolve target
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s: %v", host, err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for %s", host)
	}

	targetIP := ips[0]
	isIPv6 := targetIP.To4() == nil

	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n> Traceroute to %s (%s), 30 hops max", host, targetIP.String()))

	const maxHops = 30
	const probesPerHop = 3
	const timeout = 2 * time.Second

	reachedTarget := false

	for ttl := 1; ttl <= maxHops && !reachedTarget; ttl++ {
		hopInfo := probeHopUDP(targetIP.String(), ttl, probesPerHop, timeout, isIPv6)

		// Format hop output
		output.WriteString(fmt.Sprintf("\n> %2d  ", ttl))

		if len(hopInfo.RTTs) == 0 {
			// All probes timed out
			output.WriteString("* * *")
		} else {
			// Show hostname/IP
			if hopInfo.Hostname != "" && hopInfo.Hostname != hopInfo.IP {
				output.WriteString(fmt.Sprintf("%s (%s)", hopInfo.Hostname, hopInfo.IP))
			} else {
				output.WriteString(hopInfo.IP)
			}

			// Show RTTs
			for _, rtt := range hopInfo.RTTs {
				output.WriteString(fmt.Sprintf(" %dms", rtt.Milliseconds()))
			}

			// Add asterisks for timed out probes
			timeoutCount := probesPerHop - len(hopInfo.RTTs)
			for i := 0; i < timeoutCount; i++ {
				output.WriteString(" *")
			}

			// Check if we reached target
			if hopInfo.IP == targetIP.String() {
				reachedTarget = true
			}
		}
	}

	if !reachedTarget {
		output.WriteString("\n> Note: Target not reached within maximum hops")
	}

	return output.String(), nil
}

// probeHopUDP performs multiple UDP probes with a specific TTL
func probeHopUDP(targetIP string, ttl, probes int, timeout time.Duration, isIPv6 bool) HopInfo {
	info := HopInfo{Hop: ttl}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < probes; i++ {
		wg.Add(1)
		go func(probeNum int) {
			defer wg.Done()

			start := time.Now()
			hopIP, success := probeSingleUDP(targetIP, ttl, timeout, isIPv6, probeNum)
			rtt := time.Since(start)

			mu.Lock()
			defer mu.Unlock()

			if success && hopIP != "" {
				if info.IP == "" {
					info.IP = hopIP
					// Try reverse DNS lookup (in background to avoid blocking)
					go func(ip string) {
						if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
							mu.Lock()
							if info.Hostname == "" {
								info.Hostname = strings.TrimSuffix(names[0], ".")
							}
							mu.Unlock()
						}
					}(hopIP)
				}
				info.RTTs = append(info.RTTs, rtt)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Give DNS lookups a moment to complete
	return info
}

// probeSingleUDP performs a single UDP probe with specified TTL
// Uses IP_RECVERR and recvmsg with MSG_ERRQUEUE to receive ICMP errors without privileges
func probeSingleUDP(targetIP string, ttl int, timeout time.Duration, isIPv6 bool, seq int) (string, bool) {
	// Use high port numbers like standard traceroute (33434 + seq)
	dstPort := 33434 + seq

	// Create UDP socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return "", false
	}
	defer syscall.Close(fd)

	// Set socket options: IP_TTL and IP_RECVERR
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl); err != nil {
		return "", false
	}

	// IP_RECVERR = 11 (Linux constant for receiving ICMP errors)
	const IP_RECVERR = 11
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, IP_RECVERR, 1); err != nil {
		return "", false
	}

	// Resolve and prepare destination address
	dstAddr := &syscall.SockaddrInet4{Port: dstPort}
	ip := net.ParseIP(targetIP)
	if ip == nil {
		return "", false
	}
	copy(dstAddr.Addr[:], ip.To4())

	// Send UDP packet
	data := []byte("SmartCalc traceroute probe")
	err = syscall.Sendto(fd, data, 0, dstAddr)
	if err != nil {
		return "", false
	}

	// Set receive timeout using platform-specific function
	tv := createTimeval(timeout)
	if err := syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); err != nil {
		return "", false
	}

	// Receive with MSG_ERRQUEUE to get ICMP errors
	const MSG_ERRQUEUE = 0x2000
	buf := make([]byte, 1500)
	control := make([]byte, 1024)

	_, controllen, _, from, err := syscall.Recvmsg(fd, buf, control, MSG_ERRQUEUE)

	if err != nil {
		// No error in queue yet, wait a bit and try again
		time.Sleep(100 * time.Millisecond)
		_, controllen, _, from, err = syscall.Recvmsg(fd, buf, control, MSG_ERRQUEUE)

		if err != nil {
			// Try normal receive to see if we reached destination
			n2, _, err2 := syscall.Recvfrom(fd, buf, 0)
			if err2 == nil && n2 > 0 {
				// Got response - reached destination
				return targetIP, true
			}
			return "", false
		}
	}

	// Parse control messages to extract hop IP from ICMP error
	if controllen > 0 {
		msgs, err := syscall.ParseSocketControlMessage(control[:controllen])
		if err == nil {
			for _, m := range msgs {
				// IP_RECVERR control message
				if m.Header.Level == syscall.IPPROTO_IP && m.Header.Type == IP_RECVERR {
					// sock_extended_err structure has offender sockaddr after the error struct
					// The structure is: ee_errno(4), ee_origin(1), ee_type(1), ee_code(1), ee_pad(1),
					// ee_info(4), ee_data(4) = 16 bytes, then sockaddr_in (16 bytes for IPv4)
					if len(m.Data) >= 32 {
						// Offender address starts at byte 16, IP is at bytes 20-23
						hopIP := net.IPv4(m.Data[20], m.Data[21], m.Data[22], m.Data[23])
						return hopIP.String(), true
					}
				}
			}
		}
	}
	// Check if from address contains hop info
	if from != nil {
		if sa, ok := from.(*syscall.SockaddrInet4); ok {
			return net.IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3]).String(), true
		}
	}

	return "", false
}

// tryDirectUDP attempts a direct UDP connection when ICMP listener is not available
func tryDirectUDP(targetIP string, port, ttl int, timeout time.Duration, network string) (string, bool) {
	udpAddr, err := net.ResolveUDPAddr(network, fmt.Sprintf("%s:%d", targetIP, port))
	if err != nil {
		return "", false
	}

	udpConn, err := net.DialUDP(network, nil, udpAddr)
	if err != nil {
		return "", false
	}
	defer udpConn.Close()

	// Set TTL
	rawConn, err := udpConn.SyscallConn()
	if err != nil {
		return "", false
	}

	if network == "udp6" {
		rawConn.Control(func(fd uintptr) {
			syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)
		})
	} else {
		rawConn.Control(func(fd uintptr) {
			syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
		})
	}

	udpConn.SetWriteDeadline(time.Now().Add(timeout))
	_, err = udpConn.Write([]byte("SmartCalc traceroute probe"))
	if err != nil {
		return "", false
	}

	// Can only detect if we reached the target (no intermediate hops without ICMP)
	return targetIP, true
}
