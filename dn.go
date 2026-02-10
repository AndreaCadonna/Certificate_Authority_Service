package main

import (
	"crypto/x509/pkix"
	"fmt"
	"net"
	"strings"
)

// ParseDN parses a Distinguished Name string into a pkix.Name.
// Supported attributes: CN, O, OU, L, ST, C.
// Format: "CN=My Root CA,O=My Org,C=US"
// Enforces CON-BD-001: subject DN validation
// Enforces CON-BD-019: request subject/SAN validation
func ParseDN(dn string) (pkix.Name, error) {
	var name pkix.Name
	if strings.TrimSpace(dn) == "" {
		return name, fmt.Errorf("distinguished name cannot be empty")
	}

	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx < 0 {
			return name, fmt.Errorf("invalid DN component: %q (missing '=')", part)
		}
		attrType := strings.TrimSpace(part[:idx])
		attrValue := strings.TrimSpace(part[idx+1:])
		if attrValue == "" {
			return name, fmt.Errorf("empty value for attribute %q", attrType)
		}

		switch strings.ToUpper(attrType) {
		case "CN":
			name.CommonName = attrValue
		case "O":
			name.Organization = append(name.Organization, attrValue)
		case "OU":
			name.OrganizationalUnit = append(name.OrganizationalUnit, attrValue)
		case "L":
			name.Locality = append(name.Locality, attrValue)
		case "ST":
			name.Province = append(name.Province, attrValue)
		case "C":
			name.Country = append(name.Country, attrValue)
		default:
			return name, fmt.Errorf("unknown attribute type %q", attrType)
		}
	}

	return name, nil
}

// FormatDN formats a pkix.Name back to a DN string.
// Output order: CN, O, OU, L, ST, C. Empty fields are skipped.
func FormatDN(name pkix.Name) string {
	var parts []string
	if name.CommonName != "" {
		parts = append(parts, "CN="+name.CommonName)
	}
	for _, o := range name.Organization {
		parts = append(parts, "O="+o)
	}
	for _, ou := range name.OrganizationalUnit {
		parts = append(parts, "OU="+ou)
	}
	for _, l := range name.Locality {
		parts = append(parts, "L="+l)
	}
	for _, st := range name.Province {
		parts = append(parts, "ST="+st)
	}
	for _, c := range name.Country {
		parts = append(parts, "C="+c)
	}
	return strings.Join(parts, ",")
}

// ParseSANs parses a comma-separated SAN list string.
// Format: "DNS:example.com,DNS:www.example.com,IP:10.0.0.1"
// Enforces CON-BD-021: SAN format validation
func ParseSANs(sanList string) (dnsNames []string, ips []net.IP, err error) {
	if strings.TrimSpace(sanList) == "" {
		return nil, nil, nil
	}

	parts := strings.Split(sanList, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "DNS:") {
			dnsName := strings.TrimPrefix(part, "DNS:")
			if dnsName == "" {
				return nil, nil, fmt.Errorf("empty DNS name in SAN: %q", part)
			}
			dnsNames = append(dnsNames, dnsName)
		} else if strings.HasPrefix(part, "IP:") {
			ipStr := strings.TrimPrefix(part, "IP:")
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, nil, fmt.Errorf("invalid IP address in SAN: %q", ipStr)
			}
			ips = append(ips, ip)
		} else {
			return nil, nil, fmt.Errorf("invalid SAN format: %q (must be DNS:<name> or IP:<address>)", part)
		}
	}

	return dnsNames, ips, nil
}

// AlgoDisplayName maps CLI key algorithm flags to display names.
func AlgoDisplayName(keyAlgo string) string {
	switch keyAlgo {
	case "ecdsa-p256":
		return "ECDSA P-256"
	case "rsa-2048":
		return "RSA 2048"
	default:
		return keyAlgo
	}
}
