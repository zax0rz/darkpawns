#!/bin/bash

# Dark Pawns Security Audit Script
# Usage: ./scripts/security-audit.sh [quick|full]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}  Dark Pawns Security Audit${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_section() {
    echo -e "\n${BLUE}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_dependencies() {
    print_section "Checking Dependencies"
    
    local missing_deps=()
    
    # Check for required tools
    for cmd in go grep find; do
        if ! command -v $cmd &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        return 1
    fi
    
    print_success "All dependencies available"
}

check_go_security() {
    print_section "Go Security Checks"
    
    # Check for govulncheck
    if command -v govulncheck &> /dev/null; then
        echo "Running govulncheck..."
        if govulncheck ./... 2>/dev/null | grep -q "No vulnerabilities found"; then
            print_success "No known vulnerabilities found"
        else
            print_warning "Potential vulnerabilities found (run 'govulncheck ./...' for details)"
        fi
    else
        print_warning "govulncheck not installed (install with: go install golang.org/x/vuln/cmd/govulncheck@latest)"
    fi
    
    # Check for gosec
    if command -v gosec &> /dev/null; then
        echo "Running gosec..."
        if gosec -quiet ./... 2>/dev/null; then
            print_success "gosec passed"
        else
            print_warning "gosec found issues (run 'gosec ./...' for details)"
        fi
    else
        print_warning "gosec not installed (install with: go install github.com/securego/gosec/v2/cmd/gosec@latest)"
    fi
}

check_hardcoded_secrets() {
    print_section "Checking for Hardcoded Secrets"
    
    local patterns=(
        "password.*="
        "secret.*="
        "key.*="
        "token.*="
        "api[_-]key"
        "aws[_-]"
        "github[_-]token"
        "slack[_-]token"
        "bearer"
        "basic.*auth"
    )
    
    local found_secrets=false
    
    for pattern in "${patterns[@]}"; do
        # Search in Go files
        if grep -r -i -n --include="*.go" "$pattern" . | grep -v "test" | grep -v "example" | grep -v "REPLACE_WITH" | grep -v "your_" | grep -v "TODO" | grep -q .; then
            print_warning "Potential hardcoded secret pattern found: $pattern"
            grep -r -i -n --include="*.go" "$pattern" . | grep -v "test" | grep -v "example" | grep -v "REPLACE_WITH" | grep -v "your_" | grep -v "TODO" | head -5
            found_secrets=true
        fi
    done
    
    if [ "$found_secrets" = false ]; then
        print_success "No hardcoded secrets found"
    fi
}

check_env_files() {
    print_section "Checking Environment Files"
    
    # Check if .env exists and contains example values
    if [ -f ".env" ]; then
        if grep -q "REPLACE_WITH\|your_\|example\|test" .env; then
            print_error ".env file contains example/template values"
            grep "REPLACE_WITH\|your_\|example\|test" .env | head -5
        else
            print_success ".env file looks properly configured"
        fi
        
        # Check permissions
        local perms=$(stat -c "%a" .env 2>/dev/null || stat -f "%A" .env)
        if [ "$perms" != "600" ] && [ "$perms" != "400" ]; then
            print_warning ".env file permissions are $perms (should be 600 or 400)"
        fi
    else
        print_warning ".env file not found (using .env.example as template)"
    fi
    
    # Check .env.example
    if [ -f ".env.example" ]; then
        if grep -q "REPLACE_WITH\|your_\|example\|test" .env.example; then
            print_success ".env.example properly uses template values"
        else
            print_warning ".env.example might contain actual secrets"
        fi
    fi
}

check_input_validation() {
    print_section "Checking Input Validation"
    
    # Check for validation imports
    if grep -r "pkg/validation" --include="*.go" . | grep -q "import"; then
        print_success "Validation package imported"
    else
        print_warning "Validation package not imported in all files"
    fi
    
    # Check for SQL injection protection
    if grep -r "Query\|Exec\|Prepare" --include="*.go" . | grep -v "test" | grep -q "\?"; then
        print_success "Parameterized queries found"
    else
        print_warning "Check for parameterized SQL queries"
    fi
}

check_cors_config() {
    print_section "Checking CORS Configuration"
    
    if grep -r "CORS_ALLOWED_ORIGINS" --include="*.go" --include="*.md" . | grep -q .; then
        print_success "CORS configuration found"
    else
        print_warning "CORS configuration not documented"
    fi
    
    # Check WebSocket origin validation
    if grep -r "CheckOrigin" --include="*.go" . | grep -q .; then
        print_success "WebSocket origin validation implemented"
    else
        print_error "WebSocket origin validation missing"
    fi
}

check_jwt_implementation() {
    print_section "Checking JWT Implementation"
    
    if grep -r "jwt\|JWT" --include="*.go" . | grep -v "test" | grep -q .; then
        print_success "JWT implementation found"
        
        # Check for JWT_SECRET environment variable
        if grep -r "JWT_SECRET" --include="*.go" --include="*.md" . | grep -q .; then
            print_success "JWT_SECRET environment variable referenced"
        else
            print_warning "JWT_SECRET environment variable not documented"
        fi
    else
        print_warning "JWT implementation not found"
    fi
}

check_rate_limiting() {
    print_section "Checking Rate Limiting"
    
    if grep -r "rate.*limit\|RateLimit\|ratelimit" --include="*.go" . | grep -v "test" | grep -q .; then
        print_success "Rate limiting implementation found"
    else
        print_warning "Rate limiting implementation not found"
    fi
}

check_audit_logging() {
    print_section "Checking Audit Logging"
    
    if grep -r "audit\|Audit" --include="*.go" . | grep -v "test" | grep -q .; then
        print_success "Audit logging implementation found"
    else
        print_warning "Audit logging implementation not found"
    fi
}

check_security_headers() {
    print_section "Checking Security Headers"
    
    if grep -r "SecurityHeaders\|Content-Security-Policy\|X-Content-Type-Options" --include="*.go" . | grep -v "test" | grep -q .; then
        print_success "Security headers implementation found"
    else
        print_warning "Security headers implementation not found"
    fi
}

check_docker_security() {
    print_section "Checking Docker Security"
    
    if [ -f "Dockerfile" ]; then
        # Check for root user
        if grep -q "USER root" Dockerfile && ! grep -q "USER [0-9]" Dockerfile; then
            print_warning "Dockerfile runs as root (consider adding non-root user)"
        fi
        
        # Check for latest tag
        if grep -q "FROM.*:latest" Dockerfile; then
            print_warning "Dockerfile uses 'latest' tag (pin to specific version)"
        fi
    fi
    
    if [ -f "docker-compose.yml" ]; then
        # Check for security options
        if grep -q "read_only\|security_opt\|cap_drop" docker-compose.yml; then
            print_success "Docker Compose has security options"
        else
            print_warning "Consider adding security options to docker-compose.yml"
        fi
    fi
}

quick_audit() {
    print_header
    echo "Running quick security audit..."
    
    check_dependencies
    check_env_files
    check_hardcoded_secrets
    check_input_validation
    check_cors_config
}

full_audit() {
    print_header
    echo "Running full security audit..."
    
    check_dependencies
    check_go_security
    check_env_files
    check_hardcoded_secrets
    check_input_validation
    check_cors_config
    check_jwt_implementation
    check_rate_limiting
    check_audit_logging
    check_security_headers
    check_docker_security
}

generate_report() {
    print_section "Security Audit Summary"
    
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    local report_file="security-audit-$(date +%Y%m%d-%H%M%S).txt"
    
    echo "Security Audit Report" > "$report_file"
    echo "Generated: $timestamp" >> "$report_file"
    echo "Mode: $MODE" >> "$report_file"
    echo "========================================" >> "$report_file"
    echo "" >> "$report_file"
    
    # Capture all output
    exec 2>&1
    exec > >(tee -a "$report_file")
}

show_usage() {
    echo "Usage: $0 [MODE]"
    echo ""
    echo "Modes:"
    echo "  quick    Run quick audit (default)"
    echo "  full     Run comprehensive audit"
    echo "  report   Run full audit and generate report"
    echo ""
    echo "Examples:"
    echo "  $0 quick          # Quick security check"
    echo "  $0 full           # Comprehensive audit"
    echo "  $0 report         # Generate audit report"
}

# Parse command line arguments
MODE="quick"
GENERATE_REPORT="false"

case "${1:-quick}" in
    quick)
        MODE="quick"
        ;;
    full)
        MODE="full"
        ;;
    report)
        MODE="full"
        GENERATE_REPORT="true"
        ;;
    -h|--help)
        show_usage
        exit 0
        ;;
    *)
        print_error "Unknown mode: $1"
        show_usage
        exit 1
        ;;
esac

# Change to script directory
cd "$(dirname "$0")/.."

# Generate report if requested
if [ "$GENERATE_REPORT" = "true" ]; then
    generate_report
fi

# Run audit
case "$MODE" in
    quick)
        quick_audit
        ;;
    full)
        full_audit
        ;;
esac

print_section "Audit Complete"

if [ "$GENERATE_REPORT" = "true" ]; then
    echo -e "\n${GREEN}Report saved to: security-audit-$(date +%Y%m%d-%H%M%S).txt${NC}"
    
    echo -e "\n${YELLOW}Next Steps:${NC}"
    echo "1. Review the audit report"
    echo "2. Address any warnings or errors"
    echo "3. Schedule regular security audits"
    echo "4. Consider implementing CI/CD security scanning"
fi

echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Security is a continuous process${NC}"
echo -e "${BLUE}========================================${NC}"