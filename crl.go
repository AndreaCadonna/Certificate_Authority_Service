package main

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// CRLResult contains the results of CRL generation.
type CRLResult struct {
	ThisUpdate   time.Time
	NextUpdate   time.Time
	CRLNumber    int64
	RevokedCount int
	CRLPath      string
}

// ReasonNames maps RFC 5280 reason code integers back to display strings.
var ReasonNames = map[int]string{
	0: "unspecified",
	1: "keyCompromise",
	3: "affiliationChanged",
	4: "superseded",
	5: "cessationOfOperation",
}

// GenerateCRL generates a signed X.509 CRL v2 containing all revoked certificates.
// Enforces CON-INV-004: CA initialization prerequisite
// Enforces CON-INV-005: chain of trust integrity (CRL signed by CA key)
// Enforces CON-INV-007: CRL number monotonicity
// Enforces CON-INV-008: SHA-256 signature algorithm (explicit)
// Enforces CON-BD-010: preconditions
// Enforces CON-BD-011: postconditions
// Enforces CON-DI-004: validate-before-mutate + atomic writes (ADR-003, ADR-006)
// Enforces CON-DI-006: CRL-index consistency (all revoked, no others)
// Enforces CON-DI-009: CRL number counter consistency
// Enforces CON-DI-013: CRL structure
// Enforces CON-DI-014: system clock for timestamps
func GenerateCRL(dataDir string, nextUpdateHours int) (*CRLResult, error) {
	// VALIDATE PHASE (ADR-003)
	if !IsInitialized(dataDir) {
		return nil, fmt.Errorf("Error: CA not initialized. Run 'ca init' first.") // REQ-ER-002
	}

	caKeyPath := filepath.Join(dataDir, "ca.key")
	caCertPath := filepath.Join(dataDir, "ca.crt")
	crlnumPath := filepath.Join(dataDir, "crlnumber")
	crlPath := filepath.Join(dataDir, "ca.crl")

	caKey, err := LoadPrivateKey(caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA key: %w", err)
	}

	caCert, err := LoadCertificate(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	index, err := LoadIndex(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	crlNumber, err := ReadCounter(crlnumPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CRL number: %w", err)
	}

	// Build revoked certificate entries (CON-DI-006: exactly the revoked set)
	var revokedEntries []x509.RevocationListEntry
	for _, entry := range index {
		if entry.Status != "revoked" {
			continue
		}

		serial, err := strconv.ParseInt(entry.Serial, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse serial %s: %w", entry.Serial, err)
		}

		revokedAt, err := time.Parse(time.RFC3339, entry.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse revocation time for serial %s: %w", entry.Serial, err)
		}

		reasonCode, ok := ReasonCodes[entry.RevocationReason]
		if !ok {
			reasonCode = 0 // default to unspecified
		}

		revokedEntries = append(revokedEntries, x509.RevocationListEntry{
			SerialNumber:   big.NewInt(serial),
			RevocationTime: revokedAt,
			ReasonCode:     reasonCode,
		})
	}

	now := time.Now().UTC() // CON-DI-014: system clock
	nextUpdate := now.Add(time.Duration(nextUpdateHours) * time.Hour)

	// Build Authority Key Identifier extension (CON-DI-013)
	akiValue, err := asn1.Marshal(struct {
		KeyIdentifier []byte `asn1:"optional,tag:0"`
	}{
		KeyIdentifier: caCert.SubjectKeyId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AKI extension: %w", err)
	}

	// Build CRL template (CON-DI-013)
	template := &x509.RevocationList{
		RevokedCertificateEntries: revokedEntries,
		Number:                    big.NewInt(crlNumber), // CON-INV-007
		ThisUpdate:                now,
		NextUpdate:                nextUpdate,
		SignatureAlgorithm:        sigAlgorithm(caKey), // CON-INV-008: explicit SHA-256
		ExtraExtensions: []pkix.Extension{
			{
				Id:       asn1.ObjectIdentifier{2, 5, 29, 35}, // AuthorityKeyIdentifier OID
				Critical: false,
				Value:    akiValue,
			},
		},
	}

	// Sign CRL with CA key (CON-INV-005)
	signer, ok := caKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("CA key does not implement crypto.Signer")
	}
	crlDER, err := x509.CreateRevocationList(rand.Reader, template, caCert, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRL: %w", err)
	}

	// STAGE SUB-PHASE (ADR-006)
	tmpPaths := []string{
		crlPath + ".tmp",
		crlnumPath + ".tmp",
	}

	crlPEM := pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlDER})
	if err := os.WriteFile(crlPath+".tmp", crlPEM, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to stage CRL: %w", err)
	}

	newCRLNumData := []byte(FormatSerial(crlNumber+1) + "\n")
	if err := os.WriteFile(crlnumPath+".tmp", newCRLNumData, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to stage CRL number: %w", err)
	}

	// COMMIT SUB-PHASE (ADR-006): rename in order: ca.crl, crlnumber
	commitOrder := []struct{ tmp, final string }{
		{crlPath + ".tmp", crlPath},       // CRL updated first
		{crlnumPath + ".tmp", crlnumPath}, // Counter advanced after
	}
	for _, c := range commitOrder {
		if err := os.Rename(c.tmp, c.final); err != nil {
			cleanupTempFiles(tmpPaths)
			return nil, fmt.Errorf("failed to commit %s: %w", c.final, err)
		}
	}

	return &CRLResult{
		ThisUpdate:   now,
		NextUpdate:   nextUpdate,
		CRLNumber:    crlNumber,
		RevokedCount: len(revokedEntries),
		CRLPath:      crlPath,
	}, nil
}
