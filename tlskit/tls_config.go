package tlskit

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func CreateTLSConfiguration(certFile, keyFile, caFile string, verifyTLS bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	conf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: verifyTLS,
	}

	return conf, nil
}
