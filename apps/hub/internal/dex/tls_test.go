package dex_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dexidp/dex/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const testTimeout = 5 * time.Second

type issued struct {
	certificate *x509.Certificate
	key         *ecdsa.PrivateKey
	der         []byte
}

// issue signs a certificate with the parent, or signs it with itself when the
// parent is nil.
func issue(t *testing.T, template *x509.Certificate, parent *issued) *issued {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	signer, signerKey := template, key
	if parent != nil {
		signer, signerKey = parent.certificate, parent.key
	}

	der, err := x509.CreateCertificate(rand.Reader, template, signer, &key.PublicKey, signerKey)
	require.NoError(t, err)

	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &issued{certificate: certificate, key: key, der: der}
}

func template(serial int64, name string, usage x509.ExtKeyUsage) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
}

func writePEM(t *testing.T, path string, kind string, der []byte) {
	t.Helper()

	block := pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der})
	require.NoError(t, os.WriteFile(path, block, 0o600))
}

func tlsPair(t *testing.T, one *issued) tls.Certificate {
	t.Helper()

	return tls.Certificate{Certificate: [][]byte{one.der}, PrivateKey: one.key}
}

// startDex serves the fake on a local port behind mutual TLS, and answers the
// configuration that reaches it.
func startDex(t *testing.T, fake api.DexServer) *config.Dex {
	t.Helper()

	authorityTemplate := template(1, "test authority", x509.ExtKeyUsageAny)
	authorityTemplate.IsCA = true
	authorityTemplate.BasicConstraintsValid = true
	authorityTemplate.KeyUsage = x509.KeyUsageCertSign

	authority := issue(t, authorityTemplate, nil)

	serverTemplate := template(2, "dex", x509.ExtKeyUsageServerAuth)
	serverTemplate.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}
	server := issue(t, serverTemplate, authority)

	client := issue(t, template(3, "hub", x509.ExtKeyUsageClientAuth), authority)

	pool := x509.NewCertPool()
	pool.AddCert(authority.certificate)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{tlsPair(t, server)},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	})))
	api.RegisterDexServer(grpcServer, fake)

	go func() { _ = grpcServer.Serve(listener) }()

	t.Cleanup(grpcServer.Stop)

	directory := t.TempDir()
	cfg := &config.Dex{
		Address:  listener.Addr().String(),
		CAFile:   filepath.Join(directory, "ca.crt"),
		CertFile: filepath.Join(directory, "client.crt"),
		KeyFile:  filepath.Join(directory, "client.key"),
		Timeout:  testTimeout,
	}

	keyDER, err := x509.MarshalECPrivateKey(client.key)
	require.NoError(t, err)

	writePEM(t, cfg.CAFile, "CERTIFICATE", authority.der)
	writePEM(t, cfg.CertFile, "CERTIFICATE", client.der)
	writePEM(t, cfg.KeyFile, "EC PRIVATE KEY", keyDER)

	return cfg
}
