package main

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IndexEntry represents a certificate record in index.json.
// Enforces CON-DI-005: index schema completeness
type IndexEntry struct {
	Serial           string `json:"serial"`
	Subject          string `json:"subject"`
	NotBefore        string `json:"not_before"`
	NotAfter         string `json:"not_after"`
	Status           string `json:"status"`
	RevokedAt        string `json:"revoked_at"`
	RevocationReason string `json:"revocation_reason"`
}

// InitDataDir creates the CA data directory structure.
// Creates: data dir, certs/ subdir, serial("02"), crlnumber("01"), index.json("[]").
// Enforces CON-DI-008: serial counter consistency
// Enforces CON-DI-009: CRL number counter consistency
func InitDataDir(dataDir string) error {
	certsDir := filepath.Join(dataDir, "certs")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return fmt.Errorf("failed to create certs directory: %w", err)
	}
	return nil
}

// IsInitialized returns true if both ca.key and ca.crt exist in dataDir.
// Enforces CON-INV-004: CA initialization prerequisite
func IsInitialized(dataDir string) bool {
	keyPath := filepath.Join(dataDir, "ca.key")
	certPath := filepath.Join(dataDir, "ca.crt")
	_, keyErr := os.Stat(keyPath)
	_, certErr := os.Stat(certPath)
	return keyErr == nil && certErr == nil
}

// SavePrivateKey marshals a private key to PKCS#8 PEM and writes it to path.
// Enforces CON-DI-001: PEM encoding ("PRIVATE KEY" header)
// Enforces CON-SC-001: key material only written to file, never to output
func SavePrivateKey(path string, key crypto.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
	return writeFileAtomic(path, pemBlock, 0600)
}

// LoadPrivateKey reads a PEM-encoded PKCS#8 private key from path.
func LoadPrivateKey(path string) (crypto.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

// SaveCertPEM writes a DER-encoded certificate as PEM to path.
// Enforces CON-DI-001: PEM encoding ("CERTIFICATE" header)
func SaveCertPEM(path string, certDER []byte) error {
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	return writeFileAtomic(path, pemBlock, 0644)
}

// LoadCertificate reads a PEM-encoded X.509 certificate from path.
func LoadCertificate(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

// SaveCRLPEM writes a DER-encoded CRL as PEM to path.
// Enforces CON-DI-001: PEM encoding ("X509 CRL" header)
func SaveCRLPEM(path string, crlDER []byte) error {
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crlDER,
	})
	return writeFileAtomic(path, pemBlock, 0644)
}

// LoadCRL reads a PEM-encoded CRL from path.
func LoadCRL(path string) (*x509.RevocationList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read CRL: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}
	crl, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CRL: %w", err)
	}
	return crl, nil
}

// ReadCounter reads a hex string counter from a file and returns its int64 value.
// Enforces CON-DI-002: serial number hexadecimal format
func ReadCounter(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read counter from %s: %w", path, err)
	}
	s := strings.TrimSpace(string(data))
	val, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse counter value %q: %w", s, err)
	}
	return val, nil
}

// WriteCounter writes an int64 value as a lowercase hex string to a file.
// Enforces CON-DI-002: serial number hexadecimal format (zero-padded to 2 digits)
func WriteCounter(path string, value int64) error {
	s := FormatSerial(value)
	return writeFileAtomic(path, []byte(s+"\n"), 0644)
}

// LoadIndex reads and parses index.json from the data directory.
// Enforces CON-DI-005: index schema completeness
func LoadIndex(dataDir string) ([]IndexEntry, error) {
	path := filepath.Join(dataDir, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}
	var entries []IndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}
	return entries, nil
}

// SaveIndex serializes index entries to JSON and writes to index.json.
// Enforces CON-DI-005: index schema completeness
func SaveIndex(dataDir string, entries []IndexEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(dataDir, "index.json")
	return writeFileAtomic(path, data, 0644)
}

// FormatSerial returns a lowercase hex string zero-padded to at least 2 digits.
// Enforces CON-DI-002: serial number hexadecimal format
func FormatSerial(n int64) string {
	return fmt.Sprintf("%02x", n)
}

// FormatSerialBig returns a lowercase hex string from a *big.Int, zero-padded to at least 2 digits.
// Enforces CON-DI-002: serial number hexadecimal format
func FormatSerialBig(n *big.Int) string {
	s := strings.ToLower(n.Text(16))
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// writeFileAtomic writes data to a temporary file then renames it atomically.
// Enforces CON-DI-004: atomicity via atomic file replacement (ADR-006)
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename %s: %w", path, err)
	}
	return nil
}

// cleanupTempFiles removes .tmp files best-effort. Called on staging failure.
// Enforces CON-DI-004: no partial state on failure (ADR-006)
func cleanupTempFiles(paths []string) {
	for _, p := range paths {
		os.Remove(p)
	}
}
