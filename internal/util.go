package internal

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
)

func getKeyPair(name string) (*rsa.PrivateKey, *x509.Certificate, error) {
	keyPair, err := tls.LoadX509KeyPair("keys/"+name+".cert", "keys/"+name+".key")
	if err != nil {
		return nil, nil, err
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	return keyPair.PrivateKey.(*rsa.PrivateKey), keyPair.Leaf, nil
}
