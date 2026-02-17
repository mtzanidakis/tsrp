# Copilot Instructions for tsrp

## Project Overview

This is a Go-based HTTPS reverse proxy for Tailscale that exposes web services to your tailnet as separate devices with TLS support. It also supports Tailscale Funnel for exposing services to the public internet.

## Tech Stack

- **Language**: Go 1.26+
- **Key Dependencies**:
  - `tailscale.com` - Tailscale module (using `tsnet` subpackage for networking)
  - `github.com/caarlos0/env/v11` - Environment variable parsing
- **Container**: Docker (multi-stage build with scratch base image)

## Build and Test Commands

```bash
# Build the project
make build

# Build static binary for Linux
make build-static

# Run tests
make test
```

## Code Style Guidelines

- Follow standard Go conventions and `gofmt` formatting
- Use meaningful variable and function names
- Keep functions focused and single-purpose
- Handle errors explicitly - do not ignore them
- Use `log.Fatal` for unrecoverable startup errors
- Use `log.Printf` for runtime logging

## Project Structure

```
├── main.go              # Main application entry point
├── main_test.go         # Unit tests
├── Makefile             # Build automation
├── Dockerfile           # Multi-stage Docker build
├── docker-compose.yml   # Example deployment configuration
├── go.mod               # Go module definition
└── .github/             # GitHub workflows and configuration
```

## Configuration

The application uses environment variables for configuration:

- `BACKEND` - Backend service URL (default: `http://127.0.0.1:8080`)
- `FUNNEL` - Enable Tailscale Funnel (default: `false`)
- `HOSTNAME` - Tailnet hostname (required)
- `HTTP_PORT` - HTTP redirect port (default: `80`)
- `HTTPS_PORT` - HTTPS listener port (default: `443`)
- `STATE_DIR` - State directory (default: `/var/lib/tsrp`)
- `TS_AUTHKEY` - Tailscale auth key (required)
- `VERBOSE` - Enable verbose logging (default: `false`)

## Testing Guidelines

- Write table-driven tests where applicable
- Use `httptest` package for HTTP handler testing
- Clear environment variables with `os.Clearenv()` before config tests
- Test both default and custom configuration values

## Security Considerations

- Never commit secrets or auth keys
- Use environment variables for sensitive configuration
- The application runs in a minimal scratch container for security
