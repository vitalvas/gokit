package xnet

import "net"

// CIDRMatcher provides fast IP address matching against a set of CIDR blocks.
// It uses a radix tree internally for O(log n) lookups instead of O(n).
// This is significantly faster when checking many IPs against a large list of CIDRs.
type CIDRMatcher struct {
	ipv4Root *trieNode
	ipv6Root *trieNode
}

// trieNode represents a node in the radix tree
type trieNode struct {
	// isTerminal indicates this node represents a complete CIDR block
	isTerminal bool
	// left child (bit = 0)
	left *trieNode
	// right child (bit = 1)
	right *trieNode
}

// NewCIDRMatcher creates a new CIDR matcher from a list of networks.
// Build time is O(n * bits) where n is the number of networks.
// Subsequent lookups are O(bits) which is O(1) for fixed-size IPs.
func NewCIDRMatcher(nets []net.IPNet) *CIDRMatcher {
	matcher := &CIDRMatcher{
		ipv4Root: &trieNode{},
		ipv6Root: &trieNode{},
	}

	for _, network := range nets {
		matcher.Add(network)
	}

	return matcher
}

// NewCIDRMatcherFromStrings creates a new CIDR matcher from CIDR strings.
// Returns an error if any CIDR string is invalid.
func NewCIDRMatcherFromStrings(cidrs []string) (*CIDRMatcher, error) {
	matcher := &CIDRMatcher{
		ipv4Root: &trieNode{},
		ipv6Root: &trieNode{},
	}

	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, ErrInvalidCIDR
		}
		matcher.Add(*network)
	}

	return matcher, nil
}

// Add adds a network to the matcher.
func (m *CIDRMatcher) Add(network net.IPNet) {
	var root *trieNode
	var ip net.IP

	if network.IP.To4() != nil {
		root = m.ipv4Root
		ip = network.IP.To4()
	} else {
		root = m.ipv6Root
		ip = network.IP.To16()
	}

	ones, _ := network.Mask.Size()
	m.insert(root, ip, ones)
}

// insert adds a network to the trie
func (m *CIDRMatcher) insert(node *trieNode, ip net.IP, prefixLen int) {
	for bitPos := 0; bitPos < prefixLen; bitPos++ {
		bit := getBit(ip, bitPos)

		if bit == 0 {
			if node.left == nil {
				node.left = &trieNode{}
			}
			node = node.left
		} else {
			if node.right == nil {
				node.right = &trieNode{}
			}
			node = node.right
		}
	}

	node.isTerminal = true
}

// Contains checks if the given IP is contained in any of the CIDR blocks.
// Returns true if a match is found, false otherwise.
func (m *CIDRMatcher) Contains(ip net.IP) bool {
	if ip.To4() != nil {
		return m.search(m.ipv4Root, ip.To4(), 32)
	}
	return m.search(m.ipv6Root, ip.To16(), 128)
}

// search traverses the trie to find if the IP matches any stored network
func (m *CIDRMatcher) search(node *trieNode, ip net.IP, totalBits int) bool {
	if node == nil {
		return false
	}

	for bitPos := 0; bitPos < totalBits; bitPos++ {
		// If current node is terminal, we found a match
		if node.isTerminal {
			return true
		}

		bit := getBit(ip, bitPos)

		if bit == 0 {
			node = node.left
		} else {
			node = node.right
		}

		if node == nil {
			return false
		}
	}

	// Check if we ended on a terminal node
	return node.isTerminal
}

// getBit returns the bit at the given position in an IP address
func getBit(ip net.IP, bitPos int) byte {
	bytePos := bitPos / 8
	bitInByte := 7 - (bitPos % 8)
	return (ip[bytePos] >> bitInByte) & 1
}
