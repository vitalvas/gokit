package xnet

import (
	"bytes"
	"errors"
	"net"
	"sort"
)

var (
	ErrInvalidCIDR = errors.New("invalid CIDR notation")
)

// CIDRMerge merges a list of IP networks into the smallest possible list of CIDR blocks.
// It combines adjacent and overlapping networks for both IPv4 and IPv6.
// The function returns the optimized list of networks.
func CIDRMerge(nets []net.IPNet) []net.IPNet {
	if len(nets) == 0 {
		return []net.IPNet{}
	}

	// Separate IPv4 and IPv6 networks
	// Pre-allocate with input capacity to avoid reallocation
	ipv4Nets := make([]net.IPNet, 0, len(nets))
	ipv6Nets := make([]net.IPNet, 0, len(nets))
	for _, n := range nets {
		if n.IP.To4() != nil {
			ipv4Nets = append(ipv4Nets, n)
		} else {
			ipv6Nets = append(ipv6Nets, n)
		}
	}

	// Merge each group separately
	mergedIPv4 := mergeCIDRList(ipv4Nets)
	mergedIPv6 := mergeCIDRList(ipv6Nets)

	// Combine results
	result := make([]net.IPNet, 0, len(mergedIPv4)+len(mergedIPv6))
	result = append(result, mergedIPv4...)
	result = append(result, mergedIPv6...)

	return result
}

// CIDRMergeString merges a list of CIDR strings into the smallest possible list.
// Returns an error if any CIDR string is invalid.
func CIDRMergeString(cidrs []string) ([]string, error) {
	if len(cidrs) == 0 {
		return []string{}, nil
	}

	// Parse all CIDRs
	nets := make([]net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, ErrInvalidCIDR
		}
		nets = append(nets, *ipNet)
	}

	// Merge networks
	merged := CIDRMerge(nets)

	// Convert back to strings
	result := make([]string, len(merged))
	for i, n := range merged {
		result[i] = n.String()
	}

	return result, nil
}

// mergeCIDRList merges a list of networks of the same IP version
func mergeCIDRList(nets []net.IPNet) []net.IPNet {
	if len(nets) == 0 {
		return []net.IPNet{}
	}

	// Sort networks by IP address
	sort.Slice(nets, func(i, j int) bool {
		return bytes.Compare(nets[i].IP, nets[j].IP) < 0
	})

	// Merge overlapping and adjacent networks
	// Pre-allocate with input capacity as worst case (no merging happens)
	result := make([]net.IPNet, 1, len(nets))
	result[0] = nets[0]

	for i := 1; i < len(nets); i++ {
		current := nets[i]
		lastIdx := len(result) - 1
		last := result[lastIdx]

		// Check if current network is contained in the last merged network
		if isSubnetOf(current, last) {
			continue
		}

		// Check if last network is contained in current network
		if isSubnetOf(last, current) {
			result[lastIdx] = current
			continue
		}

		// Check if networks are adjacent or overlapping and can be merged
		merged, ok := tryMerge(last, current)
		if ok {
			result[lastIdx] = merged
			continue
		}

		// Networks cannot be merged, add current to result
		result = append(result, current)
	}

	// Try to aggregate networks into larger blocks
	result = aggregateNetworks(result)

	return result
}

// isSubnetOf checks if subnet is contained within network
func isSubnetOf(subnet, network net.IPNet) bool {
	subnetOnes, _ := subnet.Mask.Size()
	networkOnes, _ := network.Mask.Size()

	// subnet must have equal or more specific mask
	if subnetOnes < networkOnes {
		return false
	}

	return network.Contains(subnet.IP)
}

// tryMerge attempts to merge two adjacent or overlapping networks
func tryMerge(a, b net.IPNet) (net.IPNet, bool) {
	aOnes, aBits := a.Mask.Size()
	bOnes, bBits := b.Mask.Size()

	// Networks must have the same mask size to be merged
	if aOnes != bOnes || aBits != bBits {
		return net.IPNet{}, false
	}

	// Check if networks are adjacent
	if aOnes == 0 {
		return net.IPNet{}, false
	}

	// Calculate the parent network (one bit less specific)
	parentMask := net.CIDRMask(aOnes-1, aBits)
	aParent := a.IP.Mask(parentMask)
	bParent := b.IP.Mask(parentMask)

	// Networks can be merged if they share the same parent
	if !aParent.Equal(bParent) {
		return net.IPNet{}, false
	}

	// Normalize IPs to ensure we're comparing the right format
	aIP := a.IP
	bIP := b.IP
	if aBits == 32 {
		aIP = aIP.To4()
		bIP = bIP.To4()
	} else {
		aIP = aIP.To16()
		bIP = bIP.To16()
	}

	// Check if they differ only at the bit position indicated by the mask
	// For two networks to be adjacent siblings, they must differ at exactly
	// the bit position at (aOnes-1)
	bitPos := aOnes - 1
	bytePos := bitPos / 8
	bitInByte := 7 - (bitPos % 8)

	// Check if all bits before bitPos are the same
	if bytePos > 0 {
		for i := 0; i < bytePos; i++ {
			if aIP[i] != bIP[i] {
				return net.IPNet{}, false
			}
		}
	}

	// Check the byte containing the critical bit
	aBit := (aIP[bytePos] >> bitInByte) & 1
	bBit := (bIP[bytePos] >> bitInByte) & 1

	// They must have opposite bits at this position
	if aBit == bBit {
		return net.IPNet{}, false
	}

	// Check that all bits after bitPos in this byte are the same
	if bitInByte > 0 {
		mask := byte((1 << bitInByte) - 1)
		if (aIP[bytePos] & mask) != (bIP[bytePos] & mask) {
			return net.IPNet{}, false
		}
	}

	// Check if all remaining bytes are the same
	for i := bytePos + 1; i < len(aIP); i++ {
		if aIP[i] != bIP[i] {
			return net.IPNet{}, false
		}
	}

	return net.IPNet{
		IP:   aParent,
		Mask: parentMask,
	}, true
}

// aggregateNetworks tries to combine multiple networks into larger blocks
func aggregateNetworks(nets []net.IPNet) []net.IPNet {
	if len(nets) <= 1 {
		return nets
	}

	// Pre-allocate result buffers to reuse across iterations
	// Use two buffers and swap between them to avoid reallocations
	buf1 := make([]net.IPNet, 0, len(nets))
	buf2 := make([]net.IPNet, 0, len(nets))

	// Start with input in buf1
	buf1 = append(buf1, nets...)
	current := buf1
	next := buf2

	changed := true
	for changed {
		changed = false
		next = next[:0] // Reuse capacity

		for i := 0; i < len(current); i++ {
			if i+1 < len(current) {
				merged, ok := tryMerge(current[i], current[i+1])
				if ok {
					next = append(next, merged)
					i++ // Skip the next network as it's been merged
					changed = true
					continue
				}
			}
			next = append(next, current[i])
		}

		// Swap buffers for next iteration
		current, next = next, current
	}

	return current
}

// ipToInt converts an IP address to an integer for comparison
func ipToInt(ip net.IP) uint64 {
	if ip.To4() != nil {
		ip = ip.To4()
		return uint64(ip[0])<<24 | uint64(ip[1])<<16 | uint64(ip[2])<<8 | uint64(ip[3])
	}

	// For IPv6, use first 64 bits
	return uint64(ip[0])<<56 | uint64(ip[1])<<48 | uint64(ip[2])<<40 | uint64(ip[3])<<32 |
		uint64(ip[4])<<24 | uint64(ip[5])<<16 | uint64(ip[6])<<8 | uint64(ip[7])
}
