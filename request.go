package main

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"os"
)

// RequestResult contains the results of CSR generation.
type RequestResult struct {
	Subject   string
	Algorithm string
	KeyPath   string
	CSRPath   string
}

// GenerateCSR generates a key pair and PKCS#10 CSR for the ca request utility.
// Enforces CON-SC-002: cryptographically secure key generation via crypto/rand
// Enforces CON-BD-020: postconditions - PKCS#8 key, valid self-signed CSR
// Enforces CON-DI-001: PEM encoding for key and CSR
func GenerateCSR(subject pkix.Name, dnsNames []string, ips []net.IP, keyAlgo string, outKeyPath string, outCSRPath string) (*RequestResult, error) {
	// Generate key pair using CSPRNG (CON-SC-002)
	privKey, err := generateKeyPair(keyAlgo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Build CSR template
	template := &x509.CertificateRequest{
		Subject:     subject,
		DNSNames:    dnsNames,
		IPAddresses: ips,
	}

	// Create self-signed CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// Save private key as PKCS#8 PEM (CON-DI-001: "PRIVATE KEY" header)
	if err := SavePrivateKey(outKeyPath, privKey); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// Encode and save CSR as PEM (CON-DI-001: "CERTIFICATE REQUEST" header)
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})
	if err := os.WriteFile(outCSRPath, csrPEM, 0644); err != nil {
		return nil, fmt.Errorf("failed to write CSR: %w", err)
	}

	return &RequestResult{
		Subject:   FormatDN(subject),
		Algorithm: AlgoDisplayName(keyAlgo),
		KeyPath:   outKeyPath,
		CSRPath:   outCSRPath,
	}, nil
}
