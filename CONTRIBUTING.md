# Contribution

## Development

### Add a new backend service

To add a new backend service, the `TaskManager` interface must be implemented.

### Quick Commands (Makefile)

```bash
# Show all available commands
make help

# Build
make build                       # Single binary
make build-all                   # All platforms

# Testing
make test                        # Unit tests
make test-integration            # Mock backend integration tests
make test-integration-nextcloud  # Real Nextcloud sync tests
make test-all                    # All tests
make test-coverage               # Generate coverage report

# Code Quality
make lint                        # Run linter
make fmt                         # Format code
make vet                         # Run go vet
make security                    # Security scan

# Docker Test Server
make docker-up                   # Start Nextcloud server for tests
make docker-down                 # Stop server
make docker-logs                 # View logs
make docker-status               # Check status

# CI/CD
make ci                          # Run full CI suite locally
make clean                       # Clean build artifacts
make clean-all                   # Clean everything including Docker
```

### Development Workflow

1. **Fork and clone** the repository
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** with clear commit messages
4. **Run tests locally**: `make ci`
5. **Push and create a PR**

### CI/CD Pipeline

Every push and pull request triggers automated checks:

| Job | Description | Time |
|-----|-------------|------|
| **Lint** | Code quality (golangci-lint) | ~2 min |
| **Test** | Unit tests with coverage | ~2 min |
| **Integration** | Mock + Nextcloud sync tests | ~8 min |
| **Security** | Vulnerability scan (govulncheck) | ~1 min |
| **Build** | Multi-platform binaries | ~4 min |

**Branch Protection:**
- ✅ All CI checks must pass
- ✅ Code review required
- ✅ Branch must be up to date

### Running CI Locally

```bash
# Full CI suite
make ci

# Individual checks
make lint                        # Code quality
make test                        # Unit tests
make test-integration            # Integration tests
make test-integration-nextcloud  # Nextcloud tests
make security                    # Security scan
make build                       # Build check
```

### Testing Requirements

Before submitting a PR:

```bash
# Run all tests
make test-all

# Check for race conditions
go test -race ./...

# Ensure coverage doesn't drop
make test-coverage

# Format code
make fmt

# Run linter
make lint
```

### Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new backend support
fix: resolve sync conflict bug
docs: update README with examples
test: add integration tests for sync
refactor: simplify backend selection
chore: update dependencies
```

