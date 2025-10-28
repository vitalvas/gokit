package xnet

import (
	"errors"
	"net"
)

var (
	ErrInvalidPrefixSize = errors.New("new prefix must be more specific than the original network")
)

// CIDRSplit splits a network into subnets of the specified prefix length.
// For IPv4, newPrefix must be between the current prefix and 32.
// For IPv6, newPrefix must be between the current prefix and 128.
// Returns an error if the new prefix is invalid or less specific than the input.
func CIDRSplit(network net.IPNet, newPrefix int) ([]net.IPNet, error) {
	ones, bits := network.Mask.Size()

	// Validate new prefix
	if newPrefix < ones {
		return nil, ErrInvalidPrefixSize
	}

	if newPrefix > bits {
		return nil, ErrInvalidPrefixSize
	}

	// If new prefix equals current, return the network unchanged
	if newPrefix == ones {
		return []net.IPNet{network}, nil
	}

	// Calculate number of subnets
	subnetCount := 1 << uint(newPrefix-ones)

	// Pre-allocate result slice
	result := make([]net.IPNet, subnetCount)

	// Generate all subnets
	newMask := net.CIDRMask(newPrefix, bits)
	baseIP := network.IP.Mask(network.Mask)

	// Normalize IP to correct length
	if bits == 32 {
		baseIP = baseIP.To4()
	} else {
		baseIP = baseIP.To16()
	}

	for i := 0; i < subnetCount; i++ {
		// Create a copy of the base IP
		ip := make(net.IP, len(baseIP))
		copy(ip, baseIP)

		// Add offset to IP
		addToIP(ip, i, ones, newPrefix)

		result[i] = net.IPNet{
			IP:   ip,
			Mask: newMask,
		}
	}

	return result, nil
}

// CIDRSplitString splits a CIDR string into subnets of the specified prefix length.
// Returns an error if the CIDR string is invalid or the new prefix is invalid.
func CIDRSplitString(cidr string, newPrefix int) ([]string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, ErrInvalidCIDR
	}

	subnets, err := CIDRSplit(*network, newPrefix)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(subnets))
	for i, subnet := range subnets {
		result[i] = subnet.String()
	}

	return result, nil
}

// addToIP adds an offset to an IP address at the bit position specified by the prefix lengths
func addToIP(ip net.IP, offset, oldPrefix, newPrefix int) {
	// Calculate how many bits we're using for subnetting
	bitsToAdd := newPrefix - oldPrefix

	// Start from the bit position where old prefix ends
	bitPos := oldPrefix

	// Add the offset bit by bit
	for bit := bitsToAdd - 1; bit >= 0; bit-- {
		if (offset & (1 << uint(bit))) != 0 {
			setBit(ip, bitPos+(bitsToAdd-1-bit))
		}
	}
}

// setBit sets a specific bit in an IP address
func setBit(ip net.IP, bitPos int) {
	bytePos := bitPos / 8
	bitInByte := 7 - (bitPos % 8)
	ip[bytePos] |= 1 << uint(bitInByte)
}
