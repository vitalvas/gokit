package xdiscovery

type Discovery struct {
	domain string
}

func NewDiscovery(domain string) *Discovery {
	return &Discovery{
		domain: domain,
	}
}
