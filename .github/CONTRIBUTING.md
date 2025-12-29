# Contributing to gosynctasks

Thank you for your interest in contributing to gosynctasks!

## Development Setup

### Prerequisites

- Go 1.24+ (check `go.mod` for exact version)
- Docker and Docker Compose (for integration tests)
- Git

### Getting Started

1. Fork and clone the repository
```bash
git clone https://github.com/YOUR_USERNAME/gosynctasks.git
cd gosynctasks
```

2. Install dependencies
```bash
go mod download
```

3. Build the project
```bash
go build -o gosynctasks ./cmd/gosynctasks
```

4. Run tests
```bash
go test ./...
```

## Running Tests

### Unit Tests
```bash
go test -v ./...
```

### Unit Tests with Coverage
```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View coverage report
```

### Integration Tests
Integration tests require a Nextcloud test server:

```bash
# Start the test server
./scripts/start-test-server.sh

# Run integration tests
go test -v ./backend/integration_test.go
```

### Linting
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## CI/CD Pipeline

The project uses GitHub Actions for CI/CD. The pipeline runs on every push and pull request:

### Jobs

1. **Lint** - Code quality checks using golangci-lint
2. **Test** - Unit tests with race detection and coverage reporting
3. **Integration** - Integration tests with Nextcloud test server
4. **Security** - Vulnerability scanning with govulncheck
5. **Build** - Multi-platform binary builds (Linux, macOS, Windows)
6. **Release** - Automatic releases when version tags are pushed

### Workflow Files

- `.github/workflows/ci.yml` - Main CI pipeline
- `.golangci.yml` - Linter configuration

## Making Changes

### Branching Strategy

- `main` - Stable branch, protected
- `feature/*` - New features
- `fix/*` - Bug fixes
- `docs/*` - Documentation updates

### Commit Messages

Follow conventional commits format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test changes
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `perf`: Performance improvements

Examples:
```
feat(sync): add conflict resolution strategies
fix(backend): handle nil pointer in coordinator
docs(readme): update installation instructions
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Add/update tests for your changes
4. Ensure all tests pass locally
5. Update documentation if needed
6. Push your branch and create a pull request
7. Wait for CI checks to pass
8. Address review feedback
9. Squash commits if requested

### Code Review

All pull requests require:
- ✅ Passing CI checks (lint, test, build)
- ✅ Code review approval
- ✅ Up-to-date with main branch
- ✅ Adequate test coverage

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Use meaningful variable/function names
- Add comments for exported functions
- Keep functions focused and small
- Avoid unnecessary complexity

## Testing Guidelines

- Write tests for new features
- Maintain or improve code coverage
- Test edge cases and error conditions
- Use table-driven tests where appropriate
- Mock external dependencies

## Documentation

- Update README.md for user-facing changes
- Update CLAUDE.md for development context
- Add code comments for complex logic
- Update SYNC_GUIDE.md for sync-related changes

## Release Process

Releases are automated via GitHub Actions:

1. Update version in relevant files
2. Create and push a git tag:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```
3. GitHub Actions will:
   - Run all CI checks
   - Build multi-platform binaries
   - Create a GitHub release
   - Upload release artifacts

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- Be respectful and constructive in discussions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
