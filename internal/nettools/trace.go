package nettools

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// HopInfo represents information about a single hop
type HopInfo struct {
	Hop      int
	IP       string
	Hostname string
	RTTs     []time.Duration
	Timeout  bool
}

// HTTPTrace performs a TCP-based traceroute showing all hops along the path
// Uses TCP SYN packets with incrementing TTL values
//
// Note: This implementation attempts to listen for ICMP Time Exceeded messages
// to detect intermediate hops. However, this requires raw socket privileges which
// may not be available in all environments. Without these privileges, only the
// final destination will be detected, and intermediate hops will show as "*".
//
// To see all hops, run the application with appropriate network privileges
// (e.g., sudo on Linux, or grant CAP_NET_RAW capability).
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
		hopInfo := probeHopTCP(targetIP.String(), port, ttl, probesPerHop, timeout, isIPv6)

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

// probeHopTCP performs multiple TCP connection probes with a specific TTL
func probeHopTCP(targetIP string, port, ttl, probes int, timeout time.Duration, isIPv6 bool) HopInfo {
	info := HopInfo{Hop: ttl}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < probes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			hopIP, success := probeSingleTCP(targetIP, port, ttl, timeout, isIPv6)
			rtt := time.Since(start)

			mu.Lock()
			defer mu.Unlock()

			if success && hopIP != "" {
				if info.IP == "" {
					info.IP = hopIP
					// Try reverse DNS lookup
					if names, err := net.LookupAddr(hopIP); err == nil && len(names) > 0 {
						info.Hostname = strings.TrimSuffix(names[0], ".")
					}
				}
				info.RTTs = append(info.RTTs, rtt)
			}
		}()
	}

	wg.Wait()
	return info
}

// probeSingleTCP performs a single TCP probe with specified TTL
func probeSingleTCP(targetIP string, port, ttl int, timeout time.Duration, isIPv6 bool) (string, bool) {
	// Start ICMP listener to catch Time Exceeded messages
	icmpChan := make(chan string, 1)
	stopICMP := make(chan struct{})

	go listenForICMP(isIPv6, icmpChan, stopICMP)
	defer close(stopICMP)

	// Create TCP connection with TTL
	network := "tcp4"
	if isIPv6 {
		network = "tcp6"
	}

	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(network, address string, c syscall.RawConn) error {
			var err error
			c.Control(func(fd uintptr) {
				if isIPv6 {
					err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)
				} else {
					err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
				}
			})
			return err
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to connect
	connDone := make(chan struct{})
	var conn net.Conn
	var dialErr error

	go func() {
		conn, dialErr = dialer.DialContext(ctx, network, fmt.Sprintf("%s:%d", targetIP, port))
		close(connDone)
	}()

	// Wait for either connection success, ICMP response, or timeout
	select {
	case <-connDone:
		if dialErr == nil && conn != nil {
			// Successfully connected - reached target
			defer conn.Close()
			remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
			return remoteAddr.IP.String(), true
		}
		// Connection failed, check if we got ICMP
		select {
		case hopIP := <-icmpChan:
			return hopIP, true
		case <-time.After(100 * time.Millisecond):
			return "", false
		}
	case hopIP := <-icmpChan:
		// Got ICMP Time Exceeded from intermediate hop
		return hopIP, true
	case <-ctx.Done():
		// Timeout
		return "", false
	}
}

// listenForICMP listens for ICMP Time Exceeded messages
func listenForICMP(isIPv6 bool, result chan<- string, stop <-chan struct{}) {
	protocol := "ip4:icmp"
	if isIPv6 {
		protocol = "ip6:ipv6-icmp"
	}

	// Try to open ICMP listener (may fail without privileges)
	conn, err := net.ListenPacket(protocol, "")
	if err != nil {
		// Can't listen for ICMP without privileges
		// This is expected in many environments
		return
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buf := make([]byte, 1500)
	for {
		select {
		case <-stop:
			return
		default:
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				continue
			}

			// Parse ICMP packet
			if n < 8 {
				continue
			}

			// Check for Time Exceeded (Type 11)
			if buf[0] == 11 {
				// Extract source IP from ICMP packet
				if udpAddr, ok := addr.(*net.IPAddr); ok {
					select {
					case result <- udpAddr.IP.String():
					default:
					}
					return
				}
			}
		}
	}
}

// Alternative implementation using raw ICMP listener (more complex but more accurate)
func probeWithICMPListener(targetIP string, port, ttl int, timeout time.Duration, isIPv6 bool) (string, bool) {
	// This would require:
	// 1. Creating a raw ICMP socket to listen for Time Exceeded messages
	// 2. Sending TCP SYN with specific TTL
	// 3. Waiting for ICMP response or TCP response
	// 4. Parsing ICMP packet to extract hop IP
	//
	// This is significantly more complex and requires elevated privileges
	// For now, we use the simpler approach above

	network := "tcp4"
	if isIPv6 {
		network = "tcp6"
	}

	// Create connection with TTL
	var conn net.Conn
	var connErr error

	if isIPv6 {
		// IPv6
		raddr, err := net.ResolveTCPAddr(network, fmt.Sprintf("[%s]:%d", targetIP, port))
		if err != nil {
			return "", false
		}

		tcpConn, err := net.DialTCP(network, nil, raddr)
		if err != nil {
			connErr = err
		} else {
			conn = tcpConn
			// Set hop limit
			rawConn := ipv6.NewConn(tcpConn)
			rawConn.SetHopLimit(ttl)
		}
	} else {
		// IPv4
		raddr, err := net.ResolveTCPAddr(network, fmt.Sprintf("%s:%d", targetIP, port))
		if err != nil {
			return "", false
		}

		tcpConn, err := net.DialTCP(network, nil, raddr)
		if err != nil {
			connErr = err
		} else {
			conn = tcpConn
			// Set TTL
			rawConn := ipv4.NewConn(tcpConn)
			rawConn.SetTTL(ttl)
		}
	}

	if conn != nil {
		defer conn.Close()
		remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
		return remoteAddr.IP.String(), true
	}

	if connErr != nil {
		// Check if timeout
		if netErr, ok := connErr.(net.Error); ok && netErr.Timeout() {
			return "", false
		}
	}

	return "", false
}
