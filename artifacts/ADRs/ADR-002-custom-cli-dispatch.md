# ADR-002: Custom CLI Subcommand Dispatch

## Status

Accepted

## Context

The experiment requires 7 CLI subcommands (`init`, `sign`, `revoke`, `crl`, `list`, `verify`, `request`), each with different flags. Go's ecosystem offers several approaches to CLI subcommand handling:

1. The standard library `flag` package with manual subcommand dispatch.
2. Third-party frameworks like `cobra` (used by kubectl, Hugo, GitHub CLI) or `urfave/cli`.
3. The standard library `os.Args` with fully manual parsing (no flag package).

The workflow philosophy mandates zero external dependencies, boring tech, and flat file structure. SPEC.md defines exact flag names and syntax for each command.

## Decision

Use Go's standard library `flag` package with a manual `switch` statement on the subcommand name. Each subcommand creates its own `flag.FlagSet` to parse its specific flags. The subcommand is extracted from `os.Args[1]`, and a `switch` dispatches to the appropriate `run*` function.

Pattern:
```go
cmd := os.Args[1]
switch cmd {
case "init":
    os.Exit(runInit(os.Args[2:]))
case "sign":
    os.Exit(runSign(os.Args[2:]))
// ...
default:
    os.Exit(2)
}
```

Each `run*` function creates a `flag.NewFlagSet`, defines its flags, parses, validates, and calls the core logic.

## Alternatives Considered

- **cobra (github.com/spf13/cobra)**: The most popular Go CLI framework. Provides subcommand routing, flag binding, help generation, shell completion, and argument validation. Used by major Go projects. However, it is an external dependency (~5 transitive dependencies including pflag, viper). Adding cobra would violate the zero-dependency constraint (ADR-001). For 7 subcommands with straightforward flags, cobra's features (shell completion, persistent flags, command groups) are not needed. The overhead of a dependency outweighs the convenience.

- **urfave/cli (github.com/urfave/cli)**: Another popular Go CLI library with a different API style. Same problem as cobra — it is an external dependency. Adds no capability that justifies breaking the zero-dependency constraint for this experiment.

- **Fully manual os.Args parsing (no flag package)**: Parse `os.Args` directly without the `flag` package. This would work but reimplements flag parsing that the standard library already provides. The `flag` package handles edge cases (combined flags, `=` syntax, boolean flags) correctly. Rejecting it gains nothing.

## Consequences

### Positive

- Zero external dependencies maintained (consistent with ADR-001).
- The dispatch logic is approximately 30 lines of code — trivially understandable.
- Each subcommand's flags are self-contained in its `run*` function, making the code easy to navigate.
- `flag.FlagSet` with `flag.ContinueOnError` allows custom error handling for exit code 2 on parse errors (CON-BD-023).

### Negative

- No auto-generated help text. The `printUsage()` function must be maintained manually. (Acceptable for 7 commands.)
- No shell completion support. (Out of scope for an experiment.)
- Positional arguments (like `<csr-file>` in `ca sign`) must be extracted manually from `fs.Args()` after flag parsing. The `flag` package only handles named flags, not positional arguments.
- The `flag` package uses single-dash flags (`-subject`) by default, not GNU-style double-dash (`--subject`). Custom handling is needed to accept `--subject` format, or accept that Go's flag package treats `-subject` and `--subject` equivalently.

### Neutral

- The `flag` package is part of Go's standard library and is stable, well-documented, and widely understood. It is boring tech.

## References

- REQ-CL-001 through REQ-CL-009 from SPEC.md (CLI command definitions)
- CON-BD-022: Data directory resolution (implemented in resolveDataDir)
- CON-BD-023: Exit code semantics (0, 1, 2)
- ADR-001: Zero external dependencies constraint
