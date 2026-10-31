package d2dmesh

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// certName is the DNS SAN every node certificate carries and the ServerName every client
// verifies, so ordinary chain verification works whatever address a node is dialled at.
const certName = "d2dnode.mesh"

// caFromToken derives the mesh CA deterministically from the join token: every node with the
// token computes the same CA certificate, so there is nothing to distribute but the token.
// Leak of the token = anyone can mint a member certificate (docs/d2d-mesh.md section 8).
func caFromToken(token string) (*x509.Certificate, ed25519.PrivateKey, error) {
	seed := sha256.Sum256([]byte("d2dmesh-ca-v1:" + strings.TrimSpace(token)))
	key := ed25519.NewKeyFromSeed(seed[:])
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "d2dmesh CA"},
		NotBefore:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	// ed25519 signatures are deterministic, so the DER is identical on every node.
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	return cert, key, err
}

// TLSConfigs returns the server and client configs for a node: a fresh leaf certificate
// signed by the token CA, mutual verification against that CA only.
func TLSConfigs(token, nodeName string) (server, client *tls.Config, err error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil, fmt.Errorf("d2dmesh: empty join token")
	}
	ca, caKey, err := caFromToken(token)
	if err != nil {
		return nil, nil, err
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: nodeName},
		DNSNames:     []string{certName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, pub, caKey)
	if err != nil {
		return nil, nil, err
	}
	leaf := tls.Certificate{Certificate: [][]byte{der, ca.Raw}, PrivateKey: priv}
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	server = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{leaf},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}
	client = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{leaf},
		RootCAs:      pool,
		ServerName:   certName,
	}
	return server, client, nil
}
