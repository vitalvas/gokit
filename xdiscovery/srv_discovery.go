package xdiscovery

import (
	"errors"
	"net"
	"strings"
)

type SrvProtocol string

func (s *SrvProtocol) String() string {
	return string(*s)
}

const (
	SrvProtocolTCP SrvProtocol = "tcp"
	SrvProtocolUDP SrvProtocol = "udp"
	SrvProtocolTLS SrvProtocol = "tls"
)

type SrvDiscoveryOpts struct {
	Service  string
	Protocol SrvProtocol
}

type SrvDiscoveredHost struct {
	Target   string
	Port     uint16
	Weight   uint16
	Priority uint16
}

func (d *Discovery) SrvDiscovery(opts SrvDiscoveryOpts) ([]SrvDiscoveredHost, error) {
	if opts.Protocol == "" {
		opts.Protocol = SrvProtocolTCP
	}

	cname, addrs, err := net.LookupSRV(opts.Service, opts.Protocol.String(), d.domain)
	if err != nil {
		return nil, err
	}

	if cname != "" && addrs == nil {
		return nil, errors.New("cname in dns record")
	}

	resp := make([]SrvDiscoveredHost, 0, len(addrs))

	for _, row := range addrs {
		resp = append(resp, SrvDiscoveredHost{
			Target:   strings.TrimSuffix(row.Target, "."),
			Port:     row.Port,
			Weight:   row.Weight,
			Priority: row.Priority,
		})
	}

	return resp, nil
}

func (d *Discovery) SrvDiscoveryByPriority(opts SrvDiscoveryOpts) (map[uint16][]SrvDiscoveredHost, error) {
	hosts, err := d.SrvDiscovery(opts)
	if err != nil {
		return nil, err
	}

	resp := make(map[uint16][]SrvDiscoveredHost)

	for _, row := range hosts {
		resp[row.Priority] = append(resp[row.Priority], row)
	}

	return resp, nil
}
