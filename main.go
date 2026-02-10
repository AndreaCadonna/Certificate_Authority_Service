package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2) // CON-BD-023: exit code 2 for usage error
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var exitCode int
	switch cmd {
	case "init":
		exitCode = runInit(args)
	case "sign":
		exitCode = runSign(args)
	case "revoke":
		exitCode = runRevoke(args)
	case "crl":
		exitCode = runCRL(args)
	case "list":
		exitCode = runList(args)
	case "verify":
		exitCode = runVerify(args)
	case "request":
		exitCode = runRequest(args)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", cmd) // REQ-CL-009
		printUsage()
		exitCode = 2
	}

	os.Exit(exitCode)
}

// resolveDataDir implements CON-BD-022: --data-dir flag > CA_DATA_DIR env > "./ca-data"
func resolveDataDir(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envVal := os.Getenv("CA_DATA_DIR"); envVal != "" {
		return envVal
	}
	return "./ca-data"
}

// runInit handles the "ca init" command.
// Enforces CON-BD-001: precondition validation (subject required, algo valid, validity positive)
// Enforces CON-BD-023: exit codes (0 success, 1 operational, 2 usage)
// Enforces CON-SC-001: only print key file path, never key content
func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Suppress default flag error messages

	subject := fs.String("subject", "", "Distinguished Name for the root CA")
	keyAlgo := fs.String("key-algorithm", "ecdsa-p256", "Key algorithm: ecdsa-p256 or rsa-2048")
	validity := fs.Int("validity", 3650, "Validity period in days")
	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Validate required flags (CON-BD-001)
	if *subject == "" {
		fmt.Fprintln(os.Stderr, "Error: --subject is required")
		return 2
	}

	if *keyAlgo != "ecdsa-p256" && *keyAlgo != "rsa-2048" {
		fmt.Fprintf(os.Stderr, "Error: invalid key algorithm %q. Must be ecdsa-p256 or rsa-2048\n", *keyAlgo)
		return 2
	}

	if *validity <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --validity must be a positive integer")
		return 2
	}

	dir := resolveDataDir(*dataDir)

	parsedSubject, err := ParseDN(*subject)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid subject: %v\n", err)
		return 2
	}

	result, err := InitCA(dir, parsedSubject, *keyAlgo, *validity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format output per SPEC.md §4.1.1 (REQ-MK-005)
	// Enforces CON-SC-001: only print file path for key
	fmt.Println("CA initialized successfully.")
	fmt.Printf("  Subject:     %s\n", result.Subject)
	fmt.Printf("  Algorithm:   %s\n", result.Algorithm)
	fmt.Printf("  Serial:      %s\n", result.Serial)
	fmt.Printf("  Not After:   %s\n", result.NotAfter.Format(time.RFC3339))
	fmt.Printf("  Certificate: %s\n", result.CertPath)
	fmt.Printf("  Key:         %s\n", result.KeyPath)
	// REQ-MK-002: warning about unencrypted key
	fmt.Printf("Warning: CA private key is stored unencrypted at %s. Protect this file.\n", result.KeyPath)

	return 0
}

// runSign handles the "ca sign" command.
// Enforces CON-BD-004: precondition validation
// Enforces CON-BD-023: exit codes
func runSign(args []string) int {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	validity := fs.Int("validity", 365, "Validity period in days")
	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Positional argument: CSR file path
	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: CSR file path is required")
		return 2
	}
	csrFile := remaining[0]

	if *validity <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --validity must be a positive integer")
		return 2
	}

	dir := resolveDataDir(*dataDir)

	csrPEM, err := os.ReadFile(csrFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read CSR file %s: %v\n", csrFile, err)
		return 1
	}

	result, err := SignCSR(dir, csrPEM, csrFile, *validity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format output per SPEC.md §4.1.2 (REQ-MK-005)
	fmt.Println("Certificate issued successfully.")
	fmt.Printf("  Serial:      %s\n", result.Serial)
	fmt.Printf("  Subject:     %s\n", result.Subject)
	fmt.Printf("  Not After:   %s\n", result.NotAfter.Format(time.RFC3339))
	fmt.Printf("  Certificate: %s\n", result.CertPath)

	return 0
}

// runRevoke handles the "ca revoke" command.
// Enforces CON-BD-007: precondition validation
// Enforces CON-BD-023: exit codes
func runRevoke(args []string) int {
	fs := flag.NewFlagSet("revoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	reason := fs.String("reason", "unspecified", "Reason code")
	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: serial number is required")
		return 2
	}
	serialHex := strings.ToLower(remaining[0])

	// Validate reason code
	validReason := false
	for _, r := range ValidReasons {
		if *reason == r {
			validReason = true
			break
		}
	}
	if !validReason {
		fmt.Fprintf(os.Stderr, "Error: invalid reason code %q. Valid: %s\n", *reason, strings.Join(ValidReasons, ", "))
		return 2
	}

	dir := resolveDataDir(*dataDir)

	if err := RevokeCert(dir, serialHex, *reason); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format output per SPEC.md §4.1.3 (REQ-MK-005)
	fmt.Println("Certificate revoked successfully.")
	fmt.Printf("  Serial: %s\n", serialHex)
	fmt.Printf("  Reason: %s\n", *reason)

	return 0
}

// runCRL handles the "ca crl" command.
// Enforces CON-BD-010: precondition validation
// Enforces CON-BD-023: exit codes
func runCRL(args []string) int {
	fs := flag.NewFlagSet("crl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	nextUpdate := fs.Int("next-update", 24, "Hours until next CRL update")
	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	if *nextUpdate <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --next-update must be a positive integer")
		return 2
	}

	dir := resolveDataDir(*dataDir)

	result, err := GenerateCRL(dir, *nextUpdate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format output per SPEC.md §4.1.4 (REQ-MK-005)
	fmt.Println("CRL generated successfully.")
	fmt.Printf("  This Update:          %s\n", result.ThisUpdate.Format(time.RFC3339))
	fmt.Printf("  Next Update:          %s\n", result.NextUpdate.Format(time.RFC3339))
	fmt.Printf("  CRL Number:           %d\n", result.CRLNumber)
	fmt.Printf("  Revoked certificates: %d\n", result.RevokedCount)
	fmt.Printf("  CRL: %s\n", result.CRLPath)

	return 0
}

// runList handles the "ca list" command.
// Enforces CON-BD-013: precondition validation
// Enforces CON-BD-014: display status computed dynamically
// Enforces CON-BD-023: exit codes
func runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	dir := resolveDataDir(*dataDir)

	certs, err := ListCerts(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if len(certs) == 0 {
		fmt.Println("No certificates issued.")
		return 0
	}

	// Format table per SPEC.md §4.1.5
	fmt.Printf("%-8s%-9s%-22s%s\n", "SERIAL", "STATUS", "NOT AFTER", "SUBJECT")
	for _, c := range certs {
		fmt.Printf("%-8s%-9s%-22s%s\n", c.Serial, c.Status, c.NotAfter.Format(time.RFC3339), c.Subject)
	}

	return 0
}

// runVerify handles the "ca verify" command.
// Enforces CON-BD-016: precondition validation
// Enforces CON-BD-017: verification report format
// Enforces CON-BD-023: exit codes
func runVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "CA data directory path")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: certificate file path is required")
		return 2
	}
	certFile := remaining[0]

	dir := resolveDataDir(*dataDir)

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read certificate file %s: %v\n", certFile, err)
		return 1
	}

	result, err := VerifyCert(dir, certPEM, certFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format verification report per SPEC.md §4.1.6
	if result.Valid {
		fmt.Println("Certificate verification: VALID")
	} else {
		fmt.Println("Certificate verification: INVALID")
	}

	fmt.Printf("  Subject:    %s\n", result.Subject)
	fmt.Printf("  Serial:     %s\n", result.Serial)
	fmt.Printf("  Issuer:     %s\n", result.Issuer)
	fmt.Printf("  Not Before: %s\n", result.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Not After:  %s\n", result.NotAfter.Format(time.RFC3339))

	if result.SigOK {
		fmt.Println("  Signature:  OK")
	} else {
		fmt.Println("  Signature:  FAILED")
		if result.Valid {
			return 0
		}
		return 1 // Early return — no further checks shown (CON-BD-017)
	}

	if result.ExpiryOK {
		fmt.Println("  Expiry:     OK")
	} else {
		fmt.Println("  Expiry:     EXPIRED")
	}

	fmt.Printf("  Revocation: %s\n", result.RevStatus)

	if result.Valid {
		return 0
	}
	return 1
}

// runRequest handles the "ca request" command.
// Enforces CON-BD-019: precondition validation
// Enforces CON-BD-021: SAN format validation
// Enforces CON-BD-023: exit codes
func runRequest(args []string) int {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	subject := fs.String("subject", "", "Distinguished Name for the CSR")
	san := fs.String("san", "", "Comma-separated SANs: DNS:name,IP:addr")
	keyAlgo := fs.String("key-algorithm", "ecdsa-p256", "Key algorithm: ecdsa-p256 or rsa-2048")
	outKey := fs.String("out-key", "", "Output path for generated private key")
	outCSR := fs.String("out-csr", "", "Output path for generated CSR")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	// Validate required flags (CON-BD-019)
	if *subject == "" {
		fmt.Fprintln(os.Stderr, "Error: --subject is required")
		return 2
	}
	if *outKey == "" {
		fmt.Fprintln(os.Stderr, "Error: --out-key is required")
		return 2
	}
	if *outCSR == "" {
		fmt.Fprintln(os.Stderr, "Error: --out-csr is required")
		return 2
	}

	if *keyAlgo != "ecdsa-p256" && *keyAlgo != "rsa-2048" {
		fmt.Fprintf(os.Stderr, "Error: invalid key algorithm %q. Must be ecdsa-p256 or rsa-2048\n", *keyAlgo)
		return 2
	}

	parsedSubject, err := ParseDN(*subject)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid subject: %v\n", err)
		return 2
	}

	var dnsNames []string
	var ips []net.IP
	if *san != "" {
		dnsNames, ips, err = ParseSANs(*san)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid SAN: %v\n", err)
			return 2
		}
	}

	result, err := GenerateCSR(parsedSubject, dnsNames, ips, *keyAlgo, *outKey, *outCSR)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Format output per SPEC.md §4.1.7 (REQ-MK-005)
	fmt.Println("CSR generated successfully.")
	fmt.Printf("  Subject:   %s\n", result.Subject)
	fmt.Printf("  Algorithm: %s\n", result.Algorithm)
	fmt.Printf("  Key:       %s\n", result.KeyPath)
	fmt.Printf("  CSR:       %s\n", result.CSRPath)

	return 0
}

// printUsage prints available subcommands to stderr.
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: ca <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init      Initialize the root Certificate Authority")
	fmt.Fprintln(os.Stderr, "  sign      Sign a CSR and issue a certificate")
	fmt.Fprintln(os.Stderr, "  revoke    Revoke a certificate by serial number")
	fmt.Fprintln(os.Stderr, "  crl       Generate a Certificate Revocation List")
	fmt.Fprintln(os.Stderr, "  list      List all issued certificates")
	fmt.Fprintln(os.Stderr, "  verify    Verify a certificate")
	fmt.Fprintln(os.Stderr, "  request   Generate a key pair and CSR for testing")
}
