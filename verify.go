package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// VerifyResult contains the results of certificate verification.
type VerifyResult struct {
	Valid     bool
	Subject   string
	Serial    string
	Issuer    string
	NotBefore time.Time
	NotAfter  time.Time
	SigOK     bool
	SigErr    string // empty if SigOK is true
	ExpiryOK  bool
	RevStatus string // "OK (not revoked)", "REVOKED (reason: X, date: Y)", or "NOT CHECKED (no CRL available)"
}

// VerifyCert verifies a certificate's signature, validity, and revocation status.
// Enforces CON-INV-004: CA initialization prerequisite
// Enforces CON-BD-016: preconditions
// Enforces CON-BD-017: three checks in order (signature, expiry, revocation)
// Enforces CON-BD-018: error conditions
// Enforces CON-DI-014: system clock for expiry check
func VerifyCert(dataDir string, certPEM []byte, certPath string) (*VerifyResult, error) {
	// Check CA initialization (CON-INV-004)
	if !IsInitialized(dataDir) {
		return nil, fmt.Errorf("Error: CA not initialized. Run 'ca init' first.") // REQ-ER-002
	}

	// Parse the certificate to verify
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM from %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", certPath, err)
	}

	// Load CA certificate
	caCertPath := filepath.Join(dataDir, "ca.crt")
	caCert, err := LoadCertificate(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	result := &VerifyResult{
		Subject:   FormatDN(cert.Subject),
		Serial:    FormatSerialBig(cert.SerialNumber),
		Issuer:    FormatDN(cert.Issuer),
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
	}

	// Check 1: Signature validation (CON-BD-017)
	if err := cert.CheckSignatureFrom(caCert); err != nil {
		result.SigOK = false
		result.SigErr = err.Error()
		result.Valid = false
		// Early return — subsequent checks not reported if signature fails (CON-BD-017)
		return result, nil
	}
	result.SigOK = true

	// Check 2: Validity period (CON-BD-017, CON-DI-014)
	now := time.Now().UTC()
	result.ExpiryOK = !now.Before(cert.NotBefore) && !now.After(cert.NotAfter)

	// Check 3: Revocation check against CRL (CON-BD-017)
	isRevoked := false
	crlFilePath := filepath.Join(dataDir, "ca.crl")
	if _, err := os.Stat(crlFilePath); err == nil {
		crl, err := LoadCRL(crlFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load CRL: %w", err)
		}

		for _, entry := range crl.RevokedCertificateEntries {
			if entry.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				isRevoked = true
				reasonName := ReasonNames[entry.ReasonCode]
				if reasonName == "" {
					reasonName = "unspecified"
				}
				result.RevStatus = fmt.Sprintf("REVOKED (reason: %s, date: %s)",
					reasonName, entry.RevocationTime.UTC().Format(time.RFC3339))
				break
			}
		}

		if !isRevoked {
			result.RevStatus = "OK (not revoked)"
		}
	} else {
		// No CRL file — does not cause failure (CON-BD-017)
		result.RevStatus = "NOT CHECKED (no CRL available)"
	}

	// Compute overall validity (CON-BD-017)
	result.Valid = result.SigOK && result.ExpiryOK && !isRevoked

	return result, nil
}
