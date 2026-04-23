#!/bin/bash

# Dark Pawns Secret Generation Script
# Usage: ./scripts/generate-secrets.sh [jwt|encryption|all]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}  Dark Pawns Secret Generation${NC}"
    echo -e "${GREEN}========================================${NC}\n"
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

generate_jwt_secret() {
    echo "Generating JWT secret..."
    
    # Generate 32-byte random string and base64 encode
    JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || \
                 head -c 32 /dev/urandom | base64)
    
    if [ -z "$JWT_SECRET" ]; then
        print_error "Failed to generate JWT secret"
        exit 1
    fi
    
    echo "JWT_SECRET=$JWT_SECRET"
    print_success "JWT secret generated"
    
    # Save to file if requested
    if [ "$SAVE_TO_FILE" = "true" ]; then
        echo "JWT_SECRET=$JWT_SECRET" >> .env.generated
        print_success "Saved to .env.generated"
    fi
}

generate_encryption_key() {
    echo "Generating encryption key..."
    
    # Generate 32-byte random string and base64 encode
    ENCRYPTION_KEY=$(openssl rand -base64 32 2>/dev/null || \
                     head -c 32 /dev/urandom | base64)
    
    if [ -z "$ENCRYPTION_KEY" ]; then
        print_error "Failed to generate encryption key"
        exit 1
    fi
    
    echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"
    print_success "Encryption key generated"
    
    # Save to file if requested
    if [ "$SAVE_TO_FILE" = "true" ]; then
        echo "ENCRYPTION_KEY=$ENCRYPTION_KEY" >> .env.generated
        print_success "Saved to .env.generated"
    fi
}

generate_api_key() {
    echo "Generating API key..."
    
    # Generate 32-character hex string
    API_KEY=$(openssl rand -hex 16 2>/dev/null || \
              head -c 16 /dev/urandom | xxd -p)
    
    if [ -z "$API_KEY" ]; then
        print_error "Failed to generate API key"
        exit 1
    fi
    
    echo "AI_API_KEY=$API_KEY"
    print_success "API key generated"
    
    # Save to file if requested
    if [ "$SAVE_TO_FILE" = "true" ]; then
        echo "AI_API_KEY=$API_KEY" >> .env.generated
        print_success "Saved to .env.generated"
    fi
}

generate_db_password() {
    echo "Generating database password..."
    
    # Generate 24-character random password
    DB_PASSWORD=$(openssl rand -base64 18 2>/dev/null || \
                  head -c 18 /dev/urandom | base64 | tr -d '\n' | cut -c1-24)
    
    if [ -z "$DB_PASSWORD" ]; then
        print_error "Failed to generate database password"
        exit 1
    fi
    
    echo "POSTGRES_PASSWORD=$DB_PASSWORD"
    print_success "Database password generated"
    
    # Save to file if requested
    if [ "$SAVE_TO_FILE" = "true" ]; then
        echo "POSTGRES_PASSWORD=$DB_PASSWORD" >> .env.generated
        print_success "Saved to .env.generated"
    fi
}

show_usage() {
    echo "Usage: $0 [OPTIONS] [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  jwt          Generate JWT secret only"
    echo "  encryption   Generate encryption key only"
    echo "  api          Generate API key only"
    echo "  db           Generate database password only"
    echo "  all          Generate all secrets (default)"
    echo ""
    echo "Options:"
    echo "  -f, --file   Save to .env.generated file"
    echo "  -h, --help   Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Generate all secrets and display"
    echo "  $0 -f all            # Generate all and save to file"
    echo "  $0 jwt               # Generate JWT secret only"
    echo "  $0 -f encryption     # Generate encryption key and save"
}

# Parse command line arguments
COMMAND="all"
SAVE_TO_FILE="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            SAVE_TO_FILE="true"
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        jwt|encryption|api|db|all)
            COMMAND="$1"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
print_header

# Remove existing .env.generated if saving to file
if [ "$SAVE_TO_FILE" = "true" ]; then
    rm -f .env.generated
    echo "# Generated on $(date)" > .env.generated
    echo "# DO NOT COMMIT THIS FILE TO VERSION CONTROL" >> .env.generated
    echo "" >> .env.generated
fi

case $COMMAND in
    jwt)
        generate_jwt_secret
        ;;
    encryption)
        generate_encryption_key
        ;;
    api)
        generate_api_key
        ;;
    db)
        generate_db_password
        ;;
    all)
        generate_jwt_secret
        echo ""
        generate_encryption_key
        echo ""
        generate_api_key
        echo ""
        generate_db_password
        ;;
esac

if [ "$SAVE_TO_FILE" = "true" ]; then
    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}  Secrets saved to .env.generated${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    echo -e "\n${YELLOW}Important:${NC}"
    echo "1. Review the generated secrets in .env.generated"
    echo "2. Copy required values to your .env file"
    echo "3. Securely delete .env.generated after copying"
    echo "4. NEVER commit .env.generated to version control"
    
    # Set restrictive permissions
    chmod 600 .env.generated 2>/dev/null || true
fi

echo -e "\n${GREEN}Done!${NC}"