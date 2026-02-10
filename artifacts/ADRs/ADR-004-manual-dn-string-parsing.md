# ADR-004: Manual Distinguished Name String Parsing

## Status

Accepted

## Context

The CLI accepts Distinguished Names as strings via the `--subject` flag (e.g., `"CN=My Root CA,O=My Org,C=US"`). Go's `crypto/x509` package requires a `pkix.Name` struct for certificate subjects and issuers. Go's standard library provides no function to parse a DN string into a `pkix.Name` struct. The reverse direction (`pkix.Name.String()`) exists but produces RFC 2253 format which may not match the input format used in the spec.

A DN parser is needed for two commands: `ca init` (CA subject) and `ca request` (CSR subject). The parser must handle the six attribute types used in SPEC.md examples: CN (CommonName), O (Organization), OU (OrganizationalUnit), L (Locality), ST (Province), C (Country).

## Decision

Implement a custom DN parser via string splitting in `dn.go`. The parser:

1. Splits the input string on `,` (comma).
2. Trims whitespace from each component.
3. Splits each component on the first `=` (to handle values containing `=`).
4. Maps the attribute type (left side) to the corresponding `pkix.Name` field.
5. Returns an error for unknown attribute types or empty values.

A corresponding `FormatDN` function outputs `pkix.Name` fields in a fixed order (CN, O, OU, L, ST, C) to produce deterministic, spec-matching output.

## Alternatives Considered

- **Go LDAP libraries (e.g., go-ldap/ldap)**: The `go-ldap/ldap` package includes `ldap.ParseDN()` which parses RFC 4514 DN strings. It handles escaped characters, multi-valued RDNs, and hex-encoded values correctly. However, it is an external dependency, violating the zero-dependency constraint (ADR-001). The DN strings in this experiment are simple (no escaped commas, no multi-valued RDNs, no hex encoding). The full RFC 4514 parser is unnecessary. Rejected because it adds an external dependency for a feature that simple string splitting handles adequately.

- **Using pkix.Name.String() for output and manual input construction**: Go's `pkix.Name` has a `.String()` method that produces RFC 2253 formatted output. We could accept input in a different format (e.g., individual flags like `--cn`, `--org`, `--country`) and use `.String()` for display. However, the spec defines `--subject` as a single DN string (REQ-CL-001, REQ-CL-007), so parsing is unavoidable. The `.String()` output order (RFC 2253 reverse) may not match the spec's example format. A custom formatter ensures consistent output. Rejected because it doesn't eliminate the need for a parser and doesn't match spec examples.

- **Regex-based parsing**: Use a regular expression to extract key-value pairs from the DN string. For example, `([A-Z]+)=([^,]+)` would match attribute-value pairs. This works but is less readable than simple string splitting and harder to produce good error messages for malformed input. Rejected because string splitting is simpler and more maintainable.

## Consequences

### Positive

- Zero external dependencies maintained (consistent with ADR-001).
- The parser is approximately 40 lines of code — straightforward to implement and review.
- Error messages can be specific: "unknown attribute type 'X'" or "empty value for attribute 'CN'".
- `FormatDN` produces output in a deterministic order matching spec examples.

### Negative

- Does not handle RFC 4514 edge cases: escaped commas in values (e.g., `CN=Doe\, John`), multi-valued RDNs (e.g., `CN=a+CN=b`), or hex-encoded values. These are not needed for the experiment but would be needed for production use.
- The parser is not a complete DN implementation. A developer unfamiliar with this codebase might expect full RFC 4514 compliance and be surprised by the limitations.

### Neutral

- The 6 supported attribute types (CN, O, OU, L, ST, C) cover all examples in SPEC.md and the vast majority of real-world certificate DNs.

## References

- REQ-CL-001: `ca init --subject <DN>` flag definition
- REQ-CL-007: `ca request --subject <DN>` flag definition
- CON-BD-001: `--subject` flag shall contain a non-empty Distinguished Name string
- CON-BD-019: `--subject` flag for request shall contain a non-empty DN
- ADR-001: Zero external dependencies constraint
