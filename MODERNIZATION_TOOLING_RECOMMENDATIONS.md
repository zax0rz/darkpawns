# Dark Pawns Modernization Tooling Recommendations

## Immediate Actions (Week 1)

### 1. Code Quality & Linting
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create .golangci.yml
cat > .golangci.yml << 'EOF'
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gocritic
    - gofmt
    - goimports
    - revive

linters-settings:
  revive:
    rules:
      - name: exported
      - name: blank-imports
      - name: context-as-argument
      - name: context-keys-type
      - name: dot-imports
      - name: error-return
      - name: error-strings
      - name: error-naming

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl
        - goconst
EOF

# Create pre-commit hook
cat > .pre-commit-config.yaml << 'EOF'
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.60.0
    hooks:
      - id: golangci-lint
        args: [--timeout=5m]
        
  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.0
    hooks:
      - id: go-fmt
      - id: go-mod-tidy
      - id: go-unit-tests
      - id: go-build
EOF
```

### 2. Development Environment
```bash
# Create .editorconfig
cat > .editorconfig << 'EOF'
root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.py]
indent_style = space
indent_size = 4

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
EOF

# Create devcontainer.json for VS Code
cat > .devcontainer/devcontainer.json << 'EOF'
{
  "name": "Dark Pawns Development",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/python:1": {
      "version": "3.11"
    },
    "ghcr.io/devcontainers/features/node:1": {
      "version": "20"
    }
  },
  "postCreateCommand": "go mod download && pip install -r requirements.txt",
  "customizations": {
    "vscode": {
      "extensions": [
        "golang.go",
        "ms-python.python",
        "ms-azuretools.vscode-docker",
        "redhat.vscode-yaml",
        "streetsidesoftware.code-spell-checker"
      ],
      "settings": {
        "go.formatTool": "gofumpt",
        "go.lintTool": "golangci-lint",
        "go.lintFlags": ["--fast"],
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
          "source.organizeImports": true
        }
      }
    }
  },
  "forwardPorts": [8080, 5432, 6379],
  "portsAttributes": {
    "8080": {
      "label": "Dark Pawns Server",
      "onAutoForward": "notify"
    },
    "5432": {
      "label": "PostgreSQL",
      "onAutoForward": "silent"
    },
    "6379": {
      "label": "Redis",
      "onAutoForward": "silent"
    }
  }
}
EOF
```

### 3. Enhanced Makefile
```makefile
# Add to existing Makefile or create Makefile.modernization
.PHONY: help lint test build clean dev deps security-scan

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

lint: ## Run linters
	golangci-lint run ./...

test: ## Run tests with coverage
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-race: ## Run tests with race detector
	go test ./... -race

bench: ## Run benchmarks
	go test ./... -bench=. -benchmem

build: ## Build all binaries
	go build -o bin/server ./cmd/server
	go build -o bin/agentkeygen ./cmd/agentkeygen

dev: ## Start development environment
	docker-compose up -d postgres redis
	air -c .air.toml

deps: ## Update dependencies
	go mod tidy
	go mod verify

security-scan: ## Run security scans
	# Install gosec first: go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...
	trivy fs --security-checks vuln,secret,config .

coverage: test ## Generate coverage report
	@echo "Coverage: $$(go tool cover -func=coverage.out | grep total | awk '{print $$3}')"

proto: ## Generate protobuf code (if using gRPC)
	protoc --go_out=. --go-grpc_out=. proto/*.proto

mocks: ## Generate mocks
	# Install mockgen first: go install go.uber.org/mock/mockgen@latest
	mockgen -source=pkg/session/manager.go -destination=pkg/session/mocks/manager_mock.go -package=mocks

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html
	go clean -cache -testcache
```

### 4. Security Configuration
```bash
# Create security scanning CI job
cat > .github/workflows/security.yml << 'EOF'
name: Security Scanning

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday

jobs:
  security:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Run gosec
      uses: securego/gosec@master
      with:
        args: -exclude-generated ./...
    
    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'
    
    - name: Upload Trivy scan results to GitHub Security tab
      uses: github/codeql-action/upload-sarif@v3
      if: always()
      with:
        sarif_file: 'trivy-results.sarif'
    
    - name: Run OWASP Dependency Check
      uses: dependency-check/Dependency-Check_Action@main
      with:
        project: 'darkpawns'
        path: '.'
        format: 'SARIF'
    
    - name: Upload Dependency Check results
      uses: github/codeql-action/upload-sarif@v3
      if: always()
      with:
        sarif_file: 'dependency-check-report.sarif'
EOF
```

### 5. Performance Monitoring
```bash
# Create performance test suite
cat > benchmarks/performance_test.go << 'EOF'
package benchmarks

import (
	"testing"
	"github.com/zax0rz/darkpawns/pkg/session"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func BenchmarkWebSocketConnection(b *testing.B) {
	// Setup
	manager := session.NewManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test WebSocket connection creation
		conn := &mockConn{}
		manager.HandleWebSocket(conn)
	}
}

func BenchmarkCombatCalculation(b *testing.B) {
	// Setup combat scenario
	attacker := &game.Player{Level: 10, Strength: 18}
	defender := &game.Player{Level: 10, ArmorClass: 5}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test combat calculation
		CalculateHit(attacker, defender)
		CalculateDamage(attacker, defender)
	}
}

func BenchmarkDatabaseSave(b *testing.B) {
	// Setup database connection
	db := setupTestDB()
	player := createTestPlayer()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test database save performance
		db.SavePlayer(player)
	}
}
EOF
```

### 6. Structured Logging Configuration
```go
// Create pkg/logging/logger.go
package logging

import (
	"context"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func New() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true,
	})
	
	return &Logger{
		Logger: slog.New(handler),
	}
}

func (l *Logger) WithRequestID(ctx context.Context) *Logger {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return &Logger{
			Logger: l.Logger.With("request_id", requestID),
		}
	}
	return l
}

// Usage in other packages:
// logger := logging.New()
// logger.Info("Player logged in", "player", playerName, "ip", ipAddress)
```

### 7. Error Handling Patterns
```go
// Create pkg/errors/errors.go
package errors

import "fmt"

type ErrorCode string

const (
	ErrPlayerNotFound ErrorCode = "PLAYER_NOT_FOUND"
	ErrInvalidCommand ErrorCode = "INVALID_COMMAND"
	ErrDatabase       ErrorCode = "DATABASE_ERROR"
	ErrAuthentication ErrorCode = "AUTHENTICATION_FAILED"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Context map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// Usage:
// if player == nil {
//     return errors.NewAppError(
//         errors.ErrPlayerNotFound,
//         "Player not found in database",
//         nil,
//     ).WithContext("player_id", playerID)
// }
```

## Priority Implementation Order

1. **Week 1:** `.golangci.yml` + `.editorconfig` + pre-commit hooks
2. **Week 1:** Resolve circular dependencies (critical path)
3. **Week 2:** Structured logging implementation
4. **Week 2:** Enhanced error handling patterns
5. **Week 3:** Security scanning CI pipeline
6. **Week 3:** Performance benchmark tests
7. **Week 4:** Development container configuration
8. **Week 4:** Enhanced Makefile with common tasks

## Quick Start Commands

```bash
# Set up development environment
make deps          # Update dependencies
make lint          # Run linters
make test          # Run tests with coverage
make security-scan # Run security scans
make dev           # Start development environment

# Code quality checks before commit
pre-commit run --all-files

# Performance testing
go test ./benchmarks -bench=. -benchmem

# Generate documentation
godoc -http=:6060
```

## Monitoring Success

- **Code quality:** `golangci-lint` output, test coverage percentage
- **Security:** Zero critical vulnerabilities in Trivy scans
- **Performance:** Benchmark results tracked over time
- **Developer experience:** Time to first contribution, build/test times