package helmengine

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// NewClientTLSConfig builds a client TLS configuration from optional PEM
// files, matching Helm's internal tlsutil (which the SDK does not export).
// A client certificate is used only when both certFile and keyFile are set.
func NewClientTLSConfig(certFile, keyFile, caFile string, insecureSkipVerify bool) (*tls.Config, error) {
	// InsecureSkipVerify is only ever set from an explicit
	// insecure_skip_tls_verify argument, the same opt-in as the Helm CLI flag.
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // explicit opt-in, mirrors helm --insecure-skip-tls-verify
	}

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if caFile != "" {
		// The tool layer has already run the path through ValidateFilePaths.
		pem, err := os.ReadFile(filepath.Clean(caFile))
		if err != nil {
			return nil, fmt.Errorf("unable to read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("CA file contains no PEM certificates")
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}
