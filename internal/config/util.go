package config

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type KeyPairRaw struct {
	Key  string `yaml:"key"`
	Cert string `yaml:"cert"`
}

type YamlConfig struct {
	IDP KeyPairRaw `yaml:"idp"`
	SP  KeyPairRaw `yaml:"sp"`
}

func GetKeyPair(name string) (*rsa.PrivateKey, *x509.Certificate, error) {
	filename, err := filepath.Abs("./config/config.yaml")
	if err != nil {
		panic(err)
	}
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	var config YamlConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	var keyPairRaw KeyPairRaw
	if name == "sp" {
		keyPairRaw = config.SP
	} else if name == "idp" {
		keyPairRaw = config.IDP
	} else {
		panic(errors.New("unrecognized key name"))
	}

	keyPair, err := tls.X509KeyPair([]byte(keyPairRaw.Cert), []byte(keyPairRaw.Key))
	// keyPair, err := tls.LoadX509KeyPair("keys/"+name+".cert", "keys/"+name+".key")
	if err != nil {
		return nil, nil, err
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	return keyPair.PrivateKey.(*rsa.PrivateKey), keyPair.Leaf, nil
}
