package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/openimsdk/tools/errs"
	"os"
)

// decryptPEM decrypts a PEM block using a password.
func decryptPEM(data []byte, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return data, nil
	}
	b, _ := pem.Decode(data)
	d, err := x509.DecryptPEMBlock(b, passphrase)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  b.Type,
		Bytes: d,
	}), nil
}

func readEncryptablePEMBlock(path string, pwd []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decryptPEM(data, pwd)
}

// newTLSConfig setup the TLS config from general config file.
func newTLSConfig(clientCertFile, clientKeyFile, caCertFile string, keyPwd []byte, insecureSkipVerify bool) (*tls.Config, error) {
	tlsConfig := tls.Config{}

	if clientCertFile != "" && clientKeyFile != "" {
		certPEMBlock, err := os.ReadFile(clientCertFile)
		if err != nil {
			return nil, errs.WrapMsg(err, "NewTLSConfig: failed to read client cert file")
		}
		keyPEMBlock, err := readEncryptablePEMBlock(clientKeyFile, keyPwd)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil, errs.WrapMsg(err, "NewTLSConfig: failed to create X509 key pair")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if caCertFile != "" {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, errs.WrapMsg(err, "NewTLSConfig: failed to read CA cert file")
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, errors.New("NewTLSConfig: not a valid CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	tlsConfig.InsecureSkipVerify = insecureSkipVerify

	return &tlsConfig, nil
}
