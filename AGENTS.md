# AGENTS.md - Developer Guide for AI Coding Assistants

This guide provides essential information for AI coding assistants working on the rtlsdr2mqtt project.

## Project Overview

RTL-SDR to MQTT bridge for smart utility meters with Home Assistant integration. Written in Go 1.25+.
Reads smart meter data using an RTL-SDR dongle and publishes readings to MQTT.

## Build, Test, and Lint Commands

### Building
```bash
make build                    # Build for current platform
make build-all                # Build for multiple platforms (linux/darwin, amd64/arm64)
go build -o rtlsdr2mqtt ./cmd/rtlsdr2mqtt  # Direct build
```

### Testing
```bash
make test                     # Run all tests with verbose output
make test-coverage            # Run tests with coverage report (generates coverage.html)
go test ./...                 # Run all tests
go test -v ./internal/config  # Run tests in a specific package
go test -run TestDefaultValues ./internal/config  # Run a single test
```

### Linting and Formatting
```bash
make lint                     # Lint code with strict rules (via golangci-lint)
make lint-fix                 # Lint and auto-fix issues
make fmt                      # Format code (gofumpt + goimports)
make check-all                # Run fmt + lint + test (full quality check)
golangci-lint run             # Direct linting
```

### Other Commands
```bash
make clean                    # Remove build artifacts and coverage files
make deps                     # Update and download dependencies
make run                      # Run with sample config
make docker                   # Build Docker image
make docker-test              # Run integration tests with docker compose
make install-tools            # Install development tools (golangci-lint)
make setup-dev                # Complete development environment setup
```

## Code Style and Conventions

### File Structure
- `cmd/` - Application entry points (main packages)
- `internal/` - Private application code (cannot be imported by other projects)
- `pkg/` - Public library code (can be imported by other projects)
- `integration-tests/` - Integration test configurations
- `scripts/` - Build and deployment scripts

### Import Organization
```go
import (
    // Standard library imports first
    "context"
    "errors"
    "fmt"
    
    // Third-party imports second
    "github.com/eclipse/paho.mqtt.golang"
    "gopkg.in/yaml.v3"
    
    // Local imports last (prefixed with module name)
    "rtlsdr2mqtt/internal/config"
    "rtlsdr2mqtt/pkg/version"
)
```

### Indentation and Formatting
- **Go files**: Use tabs (width 4) - enforced by `gofumpt` and `goimports`
- **YAML/JSON**: 2 spaces
- **Makefiles**: Tabs (width 4)
- **Shell scripts**: 2 spaces
- **Markdown**: 2 spaces
- Line length: 140 characters maximum
- Always insert final newline, trim trailing whitespace
- Use UTF-8 encoding, LF line endings

### Naming Conventions
- **Packages**: lowercase, single word when possible (e.g., `config`, `mqtt`, `decoder`)
- **Types**: PascalCase (e.g., `Config`, `MeterConfig`, `ClientConfig`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase for local, PascalCase for exported constants
- **Constants**: PascalCase for exported, camelCase for unexported
- **Acronyms**: Preserve case in names (e.g., `MQTT`, `SDR`, `USB`, `TLS`)

### Type Definitions
```go
// Always provide package documentation
// Package config handles loading and validating the application configuration.
package config

// Document all exported types with clear descriptions
// Config represents the complete application configuration.
type Config struct {
    General GeneralConfig `json:"general" yaml:"general"`
    SDR     SDRConfig     `json:"sdr"     yaml:"sdr"`
    MQTT    MQTTConfig    `json:"mqtt"    yaml:"mqtt"`
    Meters  []MeterConfig `json:"meters"  yaml:"meters"`
}

// Use struct tags consistently: align and order (json, yaml, xml, form, binding, validate)
type MeterConfig struct {
    ID                string `json:"id"                 yaml:"id"                 default:""`
    Protocol          string `json:"protocol"           yaml:"protocol"           default:"scm+"`
    UnitOfMeasurement string `json:"unit_of_measurement" yaml:"unit_of_measurement" default:"kWh"`
}
```

### Error Handling
```go
// Define static errors at package level for better error handling
var (
    ErrConfigNotFound     = errors.New("configuration file not found")
    ErrUnsupportedFormat  = errors.New("unsupported configuration file format")
    ErrNoMetersConfigured = errors.New("at least one meter must be configured")
)

// Wrap errors with context using fmt.Errorf and %w
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to load configuration file %s: %w", path, err)
    }
    // ...
}

// Use errors.Join for combining multiple errors
func loadConfigFile(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, errors.Join(ErrFileRead, err)
    }
    // ...
}

// Always check and handle errors explicitly
if err := validateConfig(config); err != nil {
    return nil, fmt.Errorf("configuration validation failed: %w", err)
}
```

### Logging
```go
// Use structured logging with log/slog
logger.Info("Starting application", "version", version.Version)
logger.Debug("Configuration loaded", "meters", len(cfg.Meters), "mqtt_host", cfg.MQTT.Host)
logger.Error("Failed to connect", "error", err)

// Pass logger through constructors, default to slog.Default()
func NewClient(config *ClientConfig, logger *slog.Logger) (Client, error) {
    if logger == nil {
        logger = slog.Default()
    }
    // ...
}
```

### Testing
- Test files: `*_test.go` in same package
- Use table-driven tests for multiple test cases
- Test function naming: `TestFunctionName` or `TestType_Method`
- Use `t.Errorf()` for non-fatal errors, `t.Fatalf()` for fatal errors
- Define test constants at package level for reusability
```go
const (
    testVerbosity       = "info"
    testBaseTopic       = "meters"
    testStateClass      = "total_increasing"
)

func TestDefaultValues(t *testing.T) {
    config := &Config{}
    err := defaults.Set(config)
    if err != nil {
        t.Fatalf("Failed to set defaults: %v", err)
    }
    
    if config.General.Verbosity != testVerbosity {
        t.Errorf("Expected verbosity '%s', got '%s'", testVerbosity, config.General.Verbosity)
    }
}
```

### Comments and Documentation
- All exported types, functions, and constants must have doc comments
- Doc comments start with the name of the element: `// LoadConfig loads...`
- Use complete sentences with proper punctuation
- Package documentation should appear before the package declaration
- Prefer clarity over brevity in comments

### Security Considerations
- Clean file paths to prevent directory traversal: `filepath.Clean(path)`
- Validate all user inputs before processing
- Use secure defaults (e.g., TLS verification enabled by default)
- Don't log sensitive information (passwords, tokens)

## Linting Configuration

This project uses **golangci-lint** with 80+ strict linters enabled (see `.golangci.yml`). Key linters:
- Core: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`
- Error handling: `err113`, `errorlint`, `wrapcheck`
- Security: `gosec`
- Style: `revive`, `gocritic`, `gofumpt`, `goimports`
- Complexity: `gocyclo` (max 15), `nestif` (max 5)
- Performance: `prealloc`, `perfsprint`

Test files have relaxed rules for: `dupl`, `errcheck`, `forcetypeassert`, `gosec`, `noctx`, `wrapcheck`

Always run `make lint` or `make check-all` before committing code.

## Git Workflow

- Main branch: `main`
- Create feature branches from `main`
- Commit messages should be clear and descriptive
- All CI checks must pass before merging

## Dependencies

Key dependencies:
- `github.com/bemasher/rtlamr` - RTL-SDR AMR meter receiver
- `github.com/eclipse/paho.mqtt.golang` - MQTT client library
- `github.com/google/gousb` - USB device access
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/creasty/defaults` - Struct field defaults

## When Making Changes

1. Run tests after changes: `make test`
2. Ensure linting passes: `make lint`
3. For config changes, update both JSON and YAML parsing
4. For MQTT changes, test Home Assistant discovery integration
5. Update README.md if user-facing features change
6. Consider backward compatibility for configuration files
