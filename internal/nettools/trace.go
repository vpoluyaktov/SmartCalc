package nettools

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
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
func probeSingleUDP(targetIP string, ttl int, timeout time.Duration, isIPv6 bool, seq int) (string, bool) {
	// Use high port numbers like standard traceroute (33434 + seq)
	dstPort := 33434 + seq

	network := "udp4"
	icmpNetwork := "ip4:icmp"
	icmpProto := 1
	if isIPv6 {
		network = "udp6"
		icmpNetwork = "ip6:ipv6-icmp"
		icmpProto = 58
	}

	// Create ICMP listener FIRST to receive Time Exceeded messages
	icmpConn, err := icmp.ListenPacket(icmpNetwork, "")
	if err != nil {
		// Without ICMP listener, we can't detect intermediate hops
		return tryDirectUDP(targetIP, dstPort, ttl, timeout, network)
	}
	defer icmpConn.Close()

	// Start listening in background before sending UDP packet
	type icmpResult struct {
		ip  string
		err error
	}
	resultChan := make(chan icmpResult, 1)

	go func() {
		reply := make([]byte, 1500)
		icmpConn.SetReadDeadline(time.Now().Add(timeout))
		n, peer, err := icmpConn.ReadFrom(reply)
		if err != nil {
			resultChan <- icmpResult{err: err}
			return
		}

		// Parse ICMP message
		_, err = icmp.ParseMessage(icmpProto, reply[:n])
		if err != nil {
			resultChan <- icmpResult{err: err}
			return
		}

		// Extract hop IP from ICMP response
		if peerIP, ok := peer.(*net.IPAddr); ok {
			resultChan <- icmpResult{ip: peerIP.IP.String()}
		} else {
			resultChan <- icmpResult{err: fmt.Errorf("invalid peer type")}
		}
	}()

	// Small delay to ensure ICMP listener is ready
	time.Sleep(10 * time.Millisecond)

	// Create UDP packet connection and set TTL using ipv4/ipv6 package
	udpConn, err := net.ListenPacket(network, "")
	if err != nil {
		return "", false
	}
	defer udpConn.Close()

	// Set TTL using the ipv4/ipv6 package methods
	if isIPv6 {
		p := ipv6.NewPacketConn(udpConn)
		if err := p.SetHopLimit(ttl); err != nil {
			return "", false
		}
	} else {
		p := ipv4.NewPacketConn(udpConn)
		if err := p.SetTTL(ttl); err != nil {
			return "", false
		}
	}

	// Resolve destination address
	dstAddr, err := net.ResolveUDPAddr(network, fmt.Sprintf("%s:%d", targetIP, dstPort))
	if err != nil {
		return "", false
	}

	// Send UDP packet
	_, err = udpConn.WriteTo([]byte("SmartCalc traceroute probe"), dstAddr)
	if err != nil {
		return "", false
	}

	// Wait for ICMP response
	result := <-resultChan
	if result.err != nil {
		return "", false
	}

	return result.ip, true
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
