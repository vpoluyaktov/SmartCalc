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

// HTTPTrace performs an ICMP-based traceroute showing all hops along the path
// Uses ICMP Echo Request packets with incrementing TTL values
//
// Note: This implementation requires raw socket privileges (CAP_NET_RAW on Linux)
// to send ICMP packets and receive ICMP Time Exceeded messages.
// Run with sudo or grant appropriate capabilities.
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
		hopInfo := probeHopICMP(targetIP.String(), ttl, probesPerHop, timeout, isIPv6)

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

// probeHopICMP performs multiple ICMP probes with a specific TTL
func probeHopICMP(targetIP string, ttl, probes int, timeout time.Duration, isIPv6 bool) HopInfo {
	info := HopInfo{Hop: ttl}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < probes; i++ {
		wg.Add(1)
		go func(probeNum int) {
			defer wg.Done()

			start := time.Now()
			hopIP, success := probeSingleICMP(targetIP, ttl, timeout, isIPv6, probeNum)
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

// probeSingleICMP performs a single ICMP probe with specified TTL
func probeSingleICMP(targetIP string, ttl int, timeout time.Duration, isIPv6 bool, seq int) (string, bool) {
	network := "ip4:icmp"
	proto := 1 // ICMP for IPv4
	if isIPv6 {
		network = "ip6:ipv6-icmp"
		proto = 58 // ICMPv6
	}

	// Create ICMP connection
	conn, err := icmp.ListenPacket(network, "")
	if err != nil {
		// Can't create ICMP socket - likely permission issue
		return "", false
	}
	defer conn.Close()

	// Set TTL
	if isIPv6 {
		p := conn.IPv6PacketConn()
		if p != nil {
			p.SetHopLimit(ttl)
		}
	} else {
		p := conn.IPv4PacketConn()
		if p != nil {
			p.SetTTL(ttl)
		}
	}

	// Create ICMP Echo Request
	var msg icmp.Message
	if isIPv6 {
		msg = icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   syscall.Getpid() & 0xffff,
				Seq:  seq,
				Data: []byte("SmartCalc traceroute"),
			},
		}
	} else {
		msg = icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   syscall.Getpid() & 0xffff,
				Seq:  seq,
				Data: []byte("SmartCalc traceroute"),
			},
		}
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return "", false
	}

	// Send ICMP packet
	dst, err := net.ResolveIPAddr(network[:len(network)-5], targetIP)
	if err != nil {
		return "", false
	}

	_, err = conn.WriteTo(msgBytes, dst)
	if err != nil {
		return "", false
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(timeout))

	// Wait for response
	reply := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			// Timeout or error
			return "", false
		}

		// Parse ICMP message
		rm, err := icmp.ParseMessage(proto, reply[:n])
		if err != nil {
			continue
		}

		switch rm.Type {
		case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
			// Got Time Exceeded from intermediate hop
			if peerIP, ok := peer.(*net.IPAddr); ok {
				return peerIP.IP.String(), true
			}
		case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
			// Got Echo Reply - reached destination
			if peerIP, ok := peer.(*net.IPAddr); ok {
				return peerIP.IP.String(), true
			}
		case ipv4.ICMPTypeDestinationUnreachable, ipv6.ICMPTypeDestinationUnreachable:
			// Destination unreachable
			if peerIP, ok := peer.(*net.IPAddr); ok {
				return peerIP.IP.String(), true
			}
		}
	}
}
