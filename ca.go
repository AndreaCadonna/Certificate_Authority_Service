package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// InitResult contains the results of CA initialization.
type InitResult struct {
	Subject   string
	Algorithm string
	Serial    string
	NotAfter  time.Time
	CertPath  string
	KeyPath   string
}

// SignResult contains the results of signing a CSR.
type SignResult struct {
	Serial   string
	Subject  string
	NotAfter time.Time
	CertPath string
}

// CertInfo contains certificate display information for listing.
type CertInfo struct {
	Serial   string
	Subject  string
	NotAfter time.Time
	Status   string // "active", "revoked", or "expired"
}

// ReasonCodes maps reason code strings to RFC 5280 CRL reason code integers.
var ReasonCodes = map[string]int{
	"unspecified":          0,
	"keyCompromise":        1,
	"affiliationChanged":   3,
	"superseded":           4,
	"cessationOfOperation": 5,
}

// ValidReasons is the ordered list of accepted reason code strings.
var ValidReasons = []string{
	"unspecified", "keyCompromise", "affiliationChanged",
	"superseded", "cessationOfOperation",
}

// generateKeyPair generates an ECDSA P-256 or RSA 2048 key pair.
// Enforces CON-SC-002: cryptographically secure key generation via crypto/rand
// Enforces CON-INV-010: supported key algorithms only
func generateKeyPair(keyAlgo string) (crypto.PrivateKey, error) {
	switch keyAlgo {
	case "ecdsa-p256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "rsa-2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	default:
		return nil, fmt.Errorf("unsupported key algorithm: %s", keyAlgo)
	}
}

// publicKeyBytes returns the DER-encoded public key bytes for SKI computation.
func publicKeyBytes(pub crypto.PublicKey) ([]byte, error) {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		return x509.MarshalPKIXPublicKey(k)
	case *rsa.PublicKey:
		return x509.MarshalPKIXPublicKey(k)
	default:
		return nil, fmt.Errorf("unsupported public key type")
	}
}

// computeSKI computes the Subject Key Identifier as SHA-1 hash of public key.
// Per RFC 5280 method 1.
func computeSKI(pub crypto.PublicKey) ([]byte, error) {
	der, err := publicKeyBytes(pub)
	if err != nil {
		return nil, err
	}
	hash := sha1.Sum(der)
	return hash[:], nil
}

// sigAlgorithm returns the appropriate signature algorithm for the key type.
// Enforces CON-INV-008: SHA-256 signature algorithm
func sigAlgorithm(key crypto.PrivateKey) x509.SignatureAlgorithm {
	switch key.(type) {
	case *ecdsa.PrivateKey:
		return x509.ECDSAWithSHA256
	case *rsa.PrivateKey:
		return x509.SHA256WithRSA
	default:
		return x509.UnknownSignatureAlgorithm
	}
}

// publicKey extracts the public key from a private key.
func publicKey(key crypto.PrivateKey) crypto.PublicKey {
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case *rsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// InitCA initializes the root CA with key pair and self-signed certificate.
// Enforces CON-INV-006: root CA self-signed identity
// Enforces CON-INV-008: SHA-256 signature algorithm (explicit)
// Enforces CON-INV-010: supported key algorithms only
// Enforces CON-BD-001: precondition validation
// Enforces CON-BD-002: postcondition - all files created
// Enforces CON-BD-003: error if already initialized
// Enforces CON-DI-004: validate-before-mutate + atomic writes (ADR-003, ADR-006)
// Enforces CON-DI-010: X.509 version 3
// Enforces CON-DI-011: root CA certificate extensions
func InitCA(dataDir string, subject pkix.Name, keyAlgo string, validityDays int) (*InitResult, error) {
	// VALIDATE PHASE (ADR-003): all checks before any state change
	if IsInitialized(dataDir) {
		return nil, fmt.Errorf("Error: CA already initialized at %s", dataDir) // REQ-ER-005
	}

	// MUTATE PHASE
	// Generate key pair using CSPRNG (CON-SC-002)
	privKey, err := generateKeyPair(keyAlgo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	pub := publicKey(privKey)

	// Compute Subject Key Identifier (CON-DI-011)
	ski, err := computeSKI(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to compute subject key identifier: %w", err)
	}

	now := time.Now().UTC() // CON-DI-014: system clock
	notAfter := now.Add(time.Duration(validityDays) * 24 * time.Hour)

	// Build X.509v3 root CA certificate template (CON-DI-011)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), // CON-INV-002: root gets serial 01
		Subject:      subject,
		NotBefore:    now,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign, // CON-DI-011
		BasicConstraintsValid: true,
		IsCA:               true, // CON-DI-011: cA=TRUE
		SubjectKeyId:       ski,  // CON-DI-011
		SignatureAlgorithm: sigAlgorithm(privKey), // CON-INV-008: explicit SHA-256
	}

	// Self-sign: template is both template and parent (CON-INV-006)
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create data directory structure
	if err := InitDataDir(dataDir); err != nil {
		return nil, err
	}

	keyPath := filepath.Join(dataDir, "ca.key")
	certPath := filepath.Join(dataDir, "ca.crt")
	serialPath := filepath.Join(dataDir, "serial")
	crlnumPath := filepath.Join(dataDir, "crlnumber")
	indexPath := filepath.Join(dataDir, "index.json")

	// Prepare all data in memory first
	serialData := FormatSerial(2) + "\n"   // CON-DI-008: next serial is 02
	crlnumData := FormatSerial(1) + "\n"   // CON-DI-009: first CRL number is 01
	indexData := "[]\n"                     // CON-INV-009: empty index, no root cert

	// STAGE SUB-PHASE (ADR-006): write all to .tmp files
	tmpPaths := []string{
		keyPath + ".tmp",
		certPath + ".tmp",
		serialPath + ".tmp",
		crlnumPath + ".tmp",
		indexPath + ".tmp",
	}

	// Stage ca.key
	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath+".tmp", keyPEM, 0600); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to write CA key: %w", err)
	}

	// Stage ca.crt
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certPath+".tmp", certPEM, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Stage serial
	if err := os.WriteFile(serialPath+".tmp", []byte(serialData), 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to write serial: %w", err)
	}

	// Stage crlnumber
	if err := os.WriteFile(crlnumPath+".tmp", []byte(crlnumData), 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to write crlnumber: %w", err)
	}

	// Stage index.json
	if err := os.WriteFile(indexPath+".tmp", []byte(indexData), 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to write index: %w", err)
	}

	// COMMIT SUB-PHASE (ADR-006): rename in order: ca.key, ca.crt, serial, crlnumber, index.json
	commitOrder := []struct{ tmp, final string }{
		{keyPath + ".tmp", keyPath},
		{certPath + ".tmp", certPath},
		{serialPath + ".tmp", serialPath},
		{crlnumPath + ".tmp", crlnumPath},
		{indexPath + ".tmp", indexPath},
	}
	for _, c := range commitOrder {
		if err := os.Rename(c.tmp, c.final); err != nil {
			cleanupTempFiles(tmpPaths)
			return nil, fmt.Errorf("failed to commit %s: %w", c.final, err)
		}
	}

	return &InitResult{
		Subject:   FormatDN(subject),
		Algorithm: AlgoDisplayName(keyAlgo),
		Serial:    FormatSerial(1),
		NotAfter:  notAfter,
		CertPath:  certPath,
		KeyPath:   keyPath,
	}, nil
}

// SignCSR validates a CSR and issues a signed end-entity certificate.
// Enforces CON-SC-003: CSR validation gate (signature + key algo before any mutation)
// Enforces CON-INV-001: serial number uniqueness via monotonic counter
// Enforces CON-INV-002: serial number monotonicity
// Enforces CON-INV-005: chain of trust integrity (signed by CA key)
// Enforces CON-INV-008: SHA-256 signature algorithm (explicit)
// Enforces CON-INV-009: index contains only end-entity certificates
// Enforces CON-INV-011: no identity verification (CON-MK-001)
// Enforces CON-DI-004: validate-before-mutate + atomic writes (ADR-003, ADR-006)
// Enforces CON-DI-010: X.509 version 3
// Enforces CON-DI-012: end-entity certificate extensions
func SignCSR(dataDir string, csrPEM []byte, csrPath string, validityDays int) (*SignResult, error) {
	// VALIDATE PHASE (ADR-003, CON-SC-003): all checks before any mutation
	if !IsInitialized(dataDir) {
		return nil, fmt.Errorf("Error: CA not initialized. Run 'ca init' first.") // REQ-ER-002
	}

	// Parse CSR PEM
	block, _ := pem.Decode(csrPEM)
	if block == nil {
		return nil, fmt.Errorf("Error: failed to parse CSR from %s", csrPath) // REQ-ER-008
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Error: failed to parse CSR from %s", csrPath) // REQ-ER-008
	}

	// Verify CSR self-signature (CON-SC-003 check 1)
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("Error: CSR signature verification failed") // REQ-ER-001
	}

	// Check key algorithm (CON-SC-003 check 2, CON-INV-010)
	switch pub := csr.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if pub.Curve != elliptic.P256() {
			return nil, fmt.Errorf("Error: unsupported key algorithm in CSR. Supported: ECDSA P-256, RSA 2048") // REQ-ER-006
		}
	case *rsa.PublicKey:
		if pub.N.BitLen() != 2048 {
			return nil, fmt.Errorf("Error: unsupported key algorithm in CSR. Supported: ECDSA P-256, RSA 2048") // REQ-ER-006
		}
	default:
		return nil, fmt.Errorf("Error: unsupported key algorithm in CSR. Supported: ECDSA P-256, RSA 2048") // REQ-ER-006
	}

	// MUTATE PHASE
	caKeyPath := filepath.Join(dataDir, "ca.key")
	caCertPath := filepath.Join(dataDir, "ca.crt")
	serialPath := filepath.Join(dataDir, "serial")

	caKey, err := LoadPrivateKey(caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA key: %w", err)
	}

	caCert, err := LoadCertificate(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	serialVal, err := ReadCounter(serialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read serial counter: %w", err)
	}

	// Compute Subject Key Identifier for end-entity cert (CON-DI-012)
	subjectSKI, err := computeSKI(csr.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute subject key identifier: %w", err)
	}

	now := time.Now().UTC() // CON-DI-014: system clock
	notAfter := now.Add(time.Duration(validityDays) * 24 * time.Hour)

	// Determine key usage based on subject key type (CON-DI-012)
	keyUsage := x509.KeyUsageDigitalSignature
	if _, isRSA := csr.PublicKey.(*rsa.PublicKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	// Build end-entity certificate template (CON-DI-012)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serialVal), // CON-INV-001, CON-INV-002
		Subject:               csr.Subject,
		NotBefore:             now,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  false, // CON-DI-012: cA=FALSE
		SubjectKeyId:          subjectSKI,
		AuthorityKeyId:        caCert.SubjectKeyId, // CON-INV-005
		DNSNames:              csr.DNSNames,
		IPAddresses:           csr.IPAddresses,
		EmailAddresses:        csr.EmailAddresses,
		SignatureAlgorithm:    sigAlgorithm(caKey), // CON-INV-008: explicit SHA-256
	}

	// Sign with CA key (CON-INV-005)
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, csr.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	serialHex := FormatSerial(serialVal)
	certFilePath := filepath.Join(dataDir, "certs", serialHex+".pem")

	// Build new index entry (CON-DI-005, CON-DI-003)
	index, err := LoadIndex(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	newEntry := IndexEntry{
		Serial:           serialHex,
		Subject:          FormatDN(csr.Subject),
		NotBefore:        now.Format(time.RFC3339),      // CON-DI-003
		NotAfter:         notAfter.Format(time.RFC3339), // CON-DI-003
		Status:           "active",
		RevokedAt:        "",
		RevocationReason: "",
	}
	updatedIndex := append(index, newEntry)

	// Prepare all data
	certPEMData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	indexData, err := marshalIndex(updatedIndex)
	if err != nil {
		return nil, err
	}
	newSerialData := []byte(FormatSerial(serialVal+1) + "\n")

	// STAGE SUB-PHASE (ADR-006)
	tmpPaths := []string{
		serialPath + ".tmp",
		certFilePath + ".tmp",
		filepath.Join(dataDir, "index.json") + ".tmp",
	}

	if err := os.WriteFile(serialPath+".tmp", newSerialData, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to stage serial: %w", err)
	}
	if err := os.WriteFile(certFilePath+".tmp", certPEMData, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to stage certificate: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "index.json")+".tmp", indexData, 0644); err != nil {
		cleanupTempFiles(tmpPaths)
		return nil, fmt.Errorf("failed to stage index: %w", err)
	}

	// COMMIT SUB-PHASE (ADR-006): rename in order: serial, cert, index
	commitOrder := []struct{ tmp, final string }{
		{serialPath + ".tmp", serialPath},           // Prevents serial reuse (CON-INV-001)
		{certFilePath + ".tmp", certFilePath},       // Places artifact
		{filepath.Join(dataDir, "index.json") + ".tmp", filepath.Join(dataDir, "index.json")}, // Commit point
	}
	for _, c := range commitOrder {
		if err := os.Rename(c.tmp, c.final); err != nil {
			cleanupTempFiles(tmpPaths)
			return nil, fmt.Errorf("failed to commit %s: %w", c.final, err)
		}
	}

	return &SignResult{
		Serial:   serialHex,
		Subject:  FormatDN(csr.Subject),
		NotAfter: notAfter,
		CertPath: certFilePath,
	}, nil
}

// RevokeCert revokes a certificate by serial number.
// Enforces CON-INV-003: certificate state irreversibility (active → revoked only)
// Enforces CON-INV-004: CA initialization prerequisite
// Enforces CON-BD-007: precondition validation
// Enforces CON-BD-008: postcondition - status, timestamp, reason set
// Enforces CON-BD-009: error conditions
// Enforces CON-DI-004: validate-before-mutate (ADR-003)
func RevokeCert(dataDir string, serialHex string, reason string) error {
	// VALIDATE PHASE (ADR-003)
	if !IsInitialized(dataDir) {
		return fmt.Errorf("Error: CA not initialized. Run 'ca init' first.") // REQ-ER-002
	}

	index, err := LoadIndex(dataDir)
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	found := -1
	for i, entry := range index {
		if entry.Serial == serialHex {
			found = i
			break
		}
	}

	if found < 0 {
		return fmt.Errorf("Error: certificate with serial %s not found", serialHex) // REQ-ER-003
	}

	if index[found].Status == "revoked" {
		return fmt.Errorf("Error: certificate with serial %s is already revoked", serialHex) // REQ-ER-004, CON-INV-003
	}

	// MUTATE PHASE
	now := time.Now().UTC() // CON-DI-014: system clock
	index[found].Status = "revoked"
	index[found].RevokedAt = now.Format(time.RFC3339)      // CON-DI-003
	index[found].RevocationReason = reason

	// Single file mutation: writeFileAtomic handles atomicity (ADR-006)
	if err := SaveIndex(dataDir, index); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// ListCerts returns all issued certificates with computed display status.
// Enforces CON-INV-004: CA initialization prerequisite
// Enforces CON-BD-013: precondition
// Enforces CON-BD-014: display status computed dynamically, read-only
func ListCerts(dataDir string) ([]CertInfo, error) {
	if !IsInitialized(dataDir) {
		return nil, fmt.Errorf("Error: CA not initialized. Run 'ca init' first.") // REQ-ER-002
	}

	index, err := LoadIndex(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	now := time.Now().UTC() // CON-DI-014: system clock
	var certs []CertInfo
	for _, entry := range index {
		notAfter, _ := time.Parse(time.RFC3339, entry.NotAfter)

		// Compute display status (CON-BD-014)
		status := "active"
		if entry.Status == "revoked" {
			status = "revoked"
		} else if now.After(notAfter) {
			status = "expired"
		}

		certs = append(certs, CertInfo{
			Serial:   entry.Serial,
			Subject:  entry.Subject,
			NotAfter: notAfter,
			Status:   status,
		})
	}

	return certs, nil
}

// marshalIndex serializes index entries to indented JSON.
func marshalIndex(entries []IndexEntry) ([]byte, error) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}
