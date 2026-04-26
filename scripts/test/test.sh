#!/bin/bash

# Dark Pawns Comprehensive Test Runner
# Run all tests: ./test.sh
# Run specific suite: ./test.sh [unit|integration|e2e|performance|security|all]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_DIR="coverage"
TEST_RESULTS_DIR="test-results"
WORLD_DIR="${WORLD_DIR:-../darkpawns/lib}"
SERVER_PORT="${SERVER_PORT:-4350}"
SERVER_HOST="${SERVER_HOST:-localhost}"
PYTHON_VENV="${PYTHON_VENV:-venv}"

# Functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

setup_directories() {
    mkdir -p "$COVERAGE_DIR"
    mkdir -p "$TEST_RESULTS_DIR"
    mkdir -p "$TEST_RESULTS_DIR/unit"
    mkdir -p "$TEST_RESULTS_DIR/integration"
    mkdir -p "$TEST_RESULTS_DIR/e2e"
    mkdir -p "$TEST_RESULTS_DIR/performance"
    mkdir -p "$TEST_RESULTS_DIR/security"
}

setup_python_env() {
    if [ ! -d "$PYTHON_VENV" ]; then
        print_header "Setting up Python virtual environment"
        python3 -m venv "$PYTHON_VENV"
        source "$PYTHON_VENV/bin/activate"
        pip install --upgrade pip
        pip install -r requirements.txt
        pip install pytest pytest-cov pytest-asyncio pytest-html requests websocket-client
    else
        source "$PYTHON_VENV/bin/activate"
    fi
}

run_unit_tests() {
    print_header "Running Unit Tests"
    
    # Go unit tests
    print_header "Go Unit Tests"
    go test -v -cover -coverprofile="$COVERAGE_DIR/unit-go.out" ./pkg/... 2>&1 | tee "$TEST_RESULTS_DIR/unit/go-tests.log"
    
    # Generate coverage report for Go
    go tool cover -html="$COVERAGE_DIR/unit-go.out" -o "$COVERAGE_DIR/unit-go.html"
    
    # Python unit tests (if any)
    if [ -d "tests/unit/python" ]; then
        print_header "Python Unit Tests"
        cd tests/unit/python && pytest -v --cov=../../.. --cov-report=html:"../../../$COVERAGE_DIR/unit-python.html" --cov-report=term 2>&1 | tee "../../../$TEST_RESULTS_DIR/unit/python-tests.log"
        cd ../../..
    fi
    
    print_success "Unit tests completed"
}

run_integration_tests() {
    print_header "Running Integration Tests"
    
    # Check if server is running
    if ! curl -s "http://$SERVER_HOST:$SERVER_PORT/health" > /dev/null; then
        print_warning "Server not running at http://$SERVER_HOST:$SERVER_PORT"
        print_warning "Starting test server..."
        ./darkpawns -world "$WORLD_DIR" -port "$SERVER_PORT" &
        SERVER_PID=$!
        sleep 5
        
        # Wait for server to be ready
        for i in {1..30}; do
            if curl -s "http://$SERVER_HOST:$SERVER_PORT/health" > /dev/null; then
                print_success "Server started successfully"
                break
            fi
            sleep 1
            if [ $i -eq 30 ]; then
                print_error "Failed to start server"
                kill $SERVER_PID 2>/dev/null || true
                return 1
            fi
        done
    fi
    
    # Python AI integration tests
    print_header "Python AI Integration Tests"
    if [ -d "tests/integration/python" ]; then
        cd tests/integration/python && pytest -v 2>&1 | tee "../../../$TEST_RESULTS_DIR/integration/python-tests.log"
        cd ../../..
    fi
    
    # Database integration tests
    print_header "Database Integration Tests"
    go test -v ./pkg/db/... 2>&1 | tee "$TEST_RESULTS_DIR/integration/db-tests.log"
    
    # Clean up if we started the server
    if [ -n "$SERVER_PID" ]; then
        print_header "Stopping test server"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    print_success "Integration tests completed"
}

run_e2e_tests() {
    print_header "Running End-to-End Tests"
    
    # Start server
    print_header "Starting server for E2E tests"
    ./darkpawns -world "$WORLD_DIR" -port "$SERVER_PORT" &
    SERVER_PID=$!
    sleep 5
    
    # Wait for server to be ready
    for i in {1..30}; do
        if curl -s "http://$SERVER_HOST:$SERVER_PORT/health" > /dev/null; then
            print_success "Server started successfully"
            break
        fi
        sleep 1
        if [ $i -eq 30 ]; then
            print_error "Failed to start server"
            kill $SERVER_PID 2>/dev/null || true
            return 1
        fi
    done
    
    # Web client tests
    print_header "Web Client E2E Tests"
    if [ -d "tests/e2e/web" ]; then
        cd tests/e2e/web && pytest -v 2>&1 | tee "../../../$TEST_RESULTS_DIR/e2e/web-tests.log"
        cd ../../..
    fi
    
    # Telnet tests
    print_header "Telnet E2E Tests"
    if [ -d "tests/e2e/telnet" ]; then
        cd tests/e2e/telnet && pytest -v 2>&1 | tee "../../../$TEST_RESULTS_DIR/e2e/telnet-tests.log"
        cd ../../..
    fi
    
    # WebSocket tests
    print_header "WebSocket E2E Tests"
    if [ -d "tests/e2e/websocket" ]; then
        cd tests/e2e/websocket && pytest -v 2>&1 | tee "../../../$TEST_RESULTS_DIR/e2e/websocket-tests.log"
        cd ../../..
    fi
    
    # Stop server
    print_header "Stopping server"
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    print_success "E2E tests completed"
}

run_performance_tests() {
    print_header "Running Performance Tests"
    
    # Start server
    print_header "Starting server for performance tests"
    ./darkpawns -world "$WORLD_DIR" -port "$SERVER_PORT" &
    SERVER_PID=$!
    sleep 5
    
    # Wait for server to be ready
    for i in {1..30}; do
        if curl -s "http://$SERVER_HOST:$SERVER_PORT/health" > /dev/null; then
            print_success "Server started successfully"
            break
        fi
        sleep 1
        if [ $i -eq 30 ]; then
            print_error "Failed to start server"
            kill $SERVER_PID 2>/dev/null || true
            return 1
        fi
    done
    
    # Load testing
    print_header "Load Testing"
    if [ -d "tests/performance" ]; then
        cd tests/performance && python load_test.py --host "$SERVER_HOST" --port "$SERVER_PORT" 2>&1 | tee "../../$TEST_RESULTS_DIR/performance/load-test.log"
        cd ../..
    fi
    
    # Stress testing
    print_header "Stress Testing"
    if [ -d "tests/performance" ]; then
        cd tests/performance && python stress_test.py --host "$SERVER_HOST" --port "$SERVER_PORT" 2>&1 | tee "../../$TEST_RESULTS_DIR/performance/stress-test.log"
        cd ../..
    fi
    
    # Memory profiling
    print_header "Memory Profiling"
    if command -v go-torch &> /dev/null; then
        go test -bench=. -benchmem -memprofile="$COVERAGE_DIR/mem.prof" ./pkg/engine/... 2>&1 | tee "$TEST_RESULTS_DIR/performance/memory-profile.log"
        go tool pprof -svg "$COVERAGE_DIR/mem.prof" > "$COVERAGE_DIR/mem-profile.svg"
    fi
    
    # CPU profiling
    print_header "CPU Profiling"
    if command -v go-torch &> /dev/null; then
        go test -bench=. -cpuprofile="$COVERAGE_DIR/cpu.prof" ./pkg/engine/... 2>&1 | tee "$TEST_RESULTS_DIR/performance/cpu-profile.log"
        go tool pprof -svg "$COVERAGE_DIR/cpu.prof" > "$COVERAGE_DIR/cpu-profile.svg"
    fi
    
    # Stop server
    print_header "Stopping server"
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    print_success "Performance tests completed"
}

run_security_tests() {
    print_header "Running Security Tests"
    
    # Start server
    print_header "Starting server for security tests"
    ./darkpawns -world "$WORLD_DIR" -port "$SERVER_PORT" &
    SERVER_PID=$!
    sleep 5
    
    # Wait for server to be ready
    for i in {1..30}; do
        if curl -s "http://$SERVER_HOST:$SERVER_PORT/health" > /dev/null; then
            print_success "Server started successfully"
            break
        fi
        sleep 1
        if [ $i -eq 30 ]; then
            print_error "Failed to start server"
            kill $SERVER_PID 2>/dev/null || true
            return 1
        fi
    done
    
    # Penetration testing
    print_header "Penetration Testing"
    if [ -d "tests/security" ]; then
        cd tests/security && python penetration_test.py --host "$SERVER_HOST" --port "$SERVER_PORT" 2>&1 | tee "../../$TEST_RESULTS_DIR/security/penetration-test.log"
        cd ../..
    fi
    
    # Vulnerability scanning
    print_header "Vulnerability Scanning"
    if command -v nmap &> /dev/null; then
        nmap -sV -p "$SERVER_PORT" "$SERVER_HOST" 2>&1 | tee "$TEST_RESULTS_DIR/security/nmap-scan.log"
    fi
    
    # SQL injection tests
    print_header "SQL Injection Tests"
    if [ -d "tests/security" ]; then
        cd tests/security && python sql_injection_test.py --host "$SERVER_HOST" --port "$SERVER_PORT" 2>&1 | tee "../../$TEST_RESULTS_DIR/security/sql-injection-test.log"
        cd ../..
    fi
    
    # XSS tests
    print_header "XSS Tests"
    if [ -d "tests/security" ]; then
        cd tests/security && python xss_test.py --host "$SERVER_HOST" --port "$SERVER_PORT" 2>&1 | tee "../../$TEST_RESULTS_DIR/security/xss-test.log"
        cd ../..
    fi
    
    # Stop server
    print_header "Stopping server"
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    print_success "Security tests completed"
}

generate_report() {
    print_header "Generating Test Report"
    
    # Count test results
    UNIT_PASS=$(grep -c "PASS" "$TEST_RESULTS_DIR/unit/go-tests.log" 2>/dev/null || echo 0)
    UNIT_FAIL=$(grep -c "FAIL" "$TEST_RESULTS_DIR/unit/go-tests.log" 2>/dev/null || echo 0)
    
    # Generate HTML report
    cat > "$TEST_RESULTS_DIR/report.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Dark Pawns Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .summary { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .test-suite { margin: 20px 0; padding: 15px; border-left: 4px solid #007bff; }
        .pass { color: green; }
        .fail { color: red; }
        .warning { color: orange; }
        table { border-collapse: collapse; width: 100%; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Dark Pawns Test Report</h1>
    <div class="summary">
        <h2>Summary</h2>
        <p>Generated: $(date)</p>
        <p>Unit Tests: <span class="pass">$UNIT_PASS passed</span>, <span class="fail">$UNIT_FAIL failed</span></p>
    </div>
    
    <div class="test-suite">
        <h2>Test Suites</h2>
        <table>
            <tr>
                <th>Suite</th>
                <th>Status</th>
                <th>Details</th>
            </tr>
            <tr>
                <td>Unit Tests</td>
                <td class="pass">Completed</td>
                <td><a href="unit/go-tests.log">View Log</a></td>
            </tr>
            <tr>
                <td>Integration Tests</td>
                <td class="warning">Pending</td>
                <td>Requires server</td>
            </tr>
            <tr>
                <td>E2E Tests</td>
                <td class="warning">Pending</td>
                <td>Requires server</td>
            </tr>
            <tr>
                <td>Performance Tests</td>
                <td class="warning">Pending</td>
                <td>Requires server</td>
            </tr>
            <tr>
                <td>Security Tests</td>
                <td class="warning">Pending</td>
                <td>Requires server</td>
            </tr>
        </table>
    </div>
    
    <div class="test-suite">
        <h2>Coverage Reports</h2>
        <ul>
            <li><a href="../coverage/unit-go.html">Go Unit Test Coverage</a></li>
        </ul>
    </div>
</body>
</html>
EOF
    
    print_success "Report generated: $TEST_RESULTS_DIR/report.html"
}

cleanup() {
    print_header "Cleaning up"
    
    # Kill any remaining server processes
    pkill -f "darkpawns -world" 2>/dev/null || true
    
    # Remove temporary files
    rm -f darkpawns 2>/dev/null || true
    
    print_success "Cleanup completed"
}

main() {
    # Parse arguments
    TEST_SUITE="${1:-all}"
    
    # Setup
    setup_directories
    setup_python_env
    
    # Build the server
    print_header "Building Dark Pawns server"
    go build -o darkpawns ./cmd/server
    if [ $? -ne 0 ]; then
        print_error "Failed to build server"
        exit 1
    fi
    print_success "Server built successfully"
    
    # Run tests based on argument
    case "$TEST_SUITE" in
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        e2e)
            run_e2e_tests
            ;;
        performance)
            run_performance_tests
            ;;
        security)
            run_security_tests
            ;;
        all)
            run_unit_tests
            run_integration_tests
            run_e2e_tests
            run_performance_tests
            run_security_tests
            ;;
        *)
            print_error "Unknown test suite: $TEST_SUITE"
            print_error "Usage: $0 [unit|integration|e2e|performance|security|all]"
            exit 1
            ;;
    esac
    
    # Generate report
    generate_report
    
    # Cleanup
    cleanup
    
    print_header "All tests completed!"
    echo -e "${GREEN}Test results saved to: $TEST_RESULTS_DIR/${NC}"
    echo -e "${GREEN}Coverage reports saved to: $COVERAGE_DIR/${NC}"
}

# Run main function
main "$@"