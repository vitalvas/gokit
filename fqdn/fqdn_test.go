package fqdn

import (
	"reflect"
	"testing"
)

func TestGetDomainNameGuesses(t *testing.T) {
	domain := "node1.pixconf.vitalvas.dev"
	domainSlice := []string{
		"node1.pixconf.vitalvas.dev",
		"pixconf.vitalvas.dev",
		"vitalvas.dev",
		"dev",
	}

	resp := GetDomainNameGuesses(domain)

	if !reflect.DeepEqual(resp, domainSlice) {
		t.Errorf("wrong split domain, got: %#v", resp)
	}
}
