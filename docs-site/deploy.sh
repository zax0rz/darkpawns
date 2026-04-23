#!/bin/bash

# Dark Pawns Documentation Site Deployment Script
# Deploys Hugo site with content negotiation support

set -e

# Configuration
SITE_DIR="/home/zach/.openclaw/workspace/darkpawns_repo/docs-site"
PUBLIC_DIR="$SITE_DIR/public"
DEPLOY_DIR="/var/www/darkpawns/docs"
NGINX_CONF="$SITE_DIR/nginx.conf"
NGINX_SITES_AVAILABLE="/etc/nginx/sites-available/darkpawns-docs"
NGINX_SITES_ENABLED="/etc/nginx/sites-enabled/darkpawns-docs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[+]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    print_status "Checking dependencies..."
    
    # Check Hugo
    if ! command -v hugo &> /dev/null; then
        print_error "Hugo not found. Please install Hugo: https://gohugo.io/installation/"
        exit 1
    fi
    
    # Check Go (for middleware)
    if ! command -v go &> /dev/null; then
        print_warning "Go not found. Middleware compilation will be skipped."
    fi
    
    # Check nginx
    if ! command -v nginx &> /dev/null; then
        print_warning "nginx not found. Configuration will be generated but not applied."
    fi
    
    print_status "Dependencies checked."
}

build_site() {
    print_status "Building Hugo site..."
    
    cd "$SITE_DIR"
    
    # Clean previous build
    if [ -d "$PUBLIC_DIR" ]; then
        rm -rf "$PUBLIC_DIR"
    fi
    
    # Build site
    hugo --minify
    
    if [ $? -ne 0 ]; then
        print_error "Hugo build failed"
        exit 1
    fi
    
    # Generate search index
    print_status "Generating search index..."
    go run middleware.go -generate-index
    
    # Copy OpenAPI spec to public directory
    if [ -f "$SITE_DIR/content/api/openapi.json" ]; then
        mkdir -p "$PUBLIC_DIR/api"
        cp "$SITE_DIR/content/api/openapi.json" "$PUBLIC_DIR/api/"
    fi
    
    print_status "Site built successfully."
}

deploy_files() {
    print_status "Deploying files..."
    
    # Create deployment directory
    sudo mkdir -p "$DEPLOY_DIR"
    
    # Copy files
    sudo cp -r "$PUBLIC_DIR"/* "$DEPLOY_DIR"/
    
    # Set permissions
    sudo chown -R www-data:www-data "$DEPLOY_DIR"
    sudo chmod -R 755 "$DEPLOY_DIR"
    
    print_status "Files deployed to $DEPLOY_DIR"
}

configure_nginx() {
    print_status "Configuring nginx..."
    
    if [ ! -f "$NGINX_CONF" ]; then
        print_error "nginx configuration not found at $NGINX_CONF"
        return 1
    fi
    
    # Copy configuration
    sudo cp "$NGINX_CONF" "$NGINX_SITES_AVAILABLE"
    
    # Create symlink if it doesn't exist
    if [ ! -L "$NGINX_SITES_ENABLED" ]; then
        sudo ln -s "$NGINX_SITES_AVAILABLE" "$NGINX_SITES_ENABLED"
    fi
    
    # Test configuration
    sudo nginx -t
    if [ $? -ne 0 ]; then
        print_error "nginx configuration test failed"
        return 1
    fi
    
    # Reload nginx
    sudo systemctl reload nginx
    
    print_status "nginx configured and reloaded."
}

setup_ssl() {
    print_status "Setting up SSL (Let's Encrypt)..."
    
    # Check if certbot is installed
    if ! command -v certbot &> /dev/null; then
        print_warning "certbot not found. SSL setup skipped."
        print_warning "Install certbot: sudo apt install certbot python3-certbot-nginx"
        return 1
    fi
    
    # Check if we have a domain configured
    if grep -q "darkpawns.labz0rz.com" "$NGINX_SITES_AVAILABLE"; then
        print_status "Requesting SSL certificate..."
        sudo certbot --nginx -d darkpawns.labz0rz.com --non-interactive --agree-tos --email hello@labz0rz.com
        
        if [ $? -eq 0 ]; then
            print_status "SSL certificate installed."
        else
            print_warning "SSL certificate request failed. Using HTTP only."
        fi
    else
        print_warning "Domain not configured in nginx. SSL setup skipped."
    fi
}

create_systemd_service() {
    print_status "Creating systemd service for Go middleware..."
    
    SERVICE_FILE="/etc/systemd/system/darkpawns-docs.service"
    
    cat > /tmp/darkpawns-docs.service << EOF
[Unit]
Description=Dark Pawns Documentation Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=$SITE_DIR
ExecStart=/usr/local/bin/darkpawns-docs
Restart=always
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=darkpawns-docs

[Install]
WantedBy=multi-user.target
EOF
    
    sudo mv /tmp/darkpawns-docs.service "$SERVICE_FILE"
    
    # Build Go middleware if Go is available
    if command -v go &> /dev/null; then
        print_status "Building Go middleware..."
        cd "$SITE_DIR"
        go build -o /tmp/darkpawns-docs middleware.go
        sudo mv /tmp/darkpawns-docs /usr/local/bin/darkpawns-docs
        sudo chmod +x /usr/local/bin/darkpawns-docs
    fi
    
    # Enable and start service
    sudo systemctl daemon-reload
    sudo systemctl enable darkpawns-docs
    sudo systemctl start darkpawns-docs
    
    print_status "Systemd service created and started."
}

generate_readme() {
    print_status "Generating deployment README..."
    
    cat > "$SITE_DIR/DEPLOYMENT.md" << 'EOF'
# Dark Pawns Documentation Site Deployment

This documentation site is built with Hugo and supports dual rendering (HTML for humans, markdown for agents).

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 User/Agent Request              │
│           (Accept: text/html or text/markdown)  │
└──────────────────────┬──────────────────────────┘
                       │
          ┌────────────▼────────────┐
          │      nginx Reverse      │
          │         Proxy           │
          │  (Content Negotiation)  │
          └────────────┬────────────┘
                       │
          ┌────────────▼────────────┐
          │   Hugo Static Site      │
          │   (HTML + Markdown)     │
          └────────────┬────────────┘
                       │
          ┌────────────▼────────────┐
          │   Go Middleware         │
          │   (Optional - dynamic   │
          │    content generation)  │
          └─────────────────────────┘
```

## Deployment Steps

### 1. Manual Deployment

```bash
# Build the site
cd /home/zach/.openclaw/workspace/darkpawns_repo/docs-site
hugo --minify

# Deploy to web server
sudo mkdir -p /var/www/darkpawns/docs
sudo cp -r public/* /var/www/darkpawns/docs/
sudo chown -R www-data:www-data /var/www/darkpawns/docs
```

### 2. Using the Deployment Script

```bash
# Make script executable
chmod +x deploy.sh

# Run deployment
./deploy.sh
```

The script will:
1. Check dependencies (Hugo, Go, nginx)
2. Build the Hugo site with minification
3. Generate search index
4. Deploy files to `/var/www/darkpawns/docs`
5. Configure nginx with content negotiation
6. Set up SSL with Let's Encrypt (if domain configured)
7. Create systemd service for Go middleware

### 3. Content Negotiation

The site supports multiple formats via HTTP `Accept` header:

```bash
# Get HTML (default)
curl https://darkpawns.labz0rz.com/docs/

# Get Markdown for agents
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/docs/

# Get search index (JSON)
curl https://darkpawns.labz0rz.com/docs/search-index.json

# Get OpenAPI spec
curl https://darkpawns.labz0rz.com/docs/api/openapi.json
```

### 4. Local Development

```bash
# Start Hugo development server
cd docs-site
hugo server -D

# Test content negotiation locally
curl -H "Accept: text/markdown" http://localhost:1313/
```

## Configuration Files

- `hugo.toml` - Hugo site configuration
- `nginx.conf` - nginx configuration with content negotiation
- `middleware.go` - Go middleware for dynamic content
- `deploy.sh` - Deployment script

## Directory Structure

```
docs-site/
├── content/                 # Markdown content
│   ├── _index.md           # Home page
│   ├── getting-started/    # Getting started guide
│   ├── agents/             # Agent documentation
│   ├── api/                # API reference
│   └── development/        # Development guide
├── themes/darkpawns-docs/  # Custom theme
│   ├── layouts/            # HTML templates
│   ├── assets/             # CSS and JS
│   └── static/             # Static files
├── public/                 # Generated site (do not edit)
├── middleware.go           # Content negotiation middleware
├── nginx.conf             # Web server configuration
└── deploy.sh              # Deployment script
```

## Search Functionality

The site includes full-text search that works for both humans and agents:

1. **Search Index**: Generated at build time (`search-index.json`)
2. **Frontend Search**: JavaScript-based search in the sidebar
3. **API Access**: Agents can download and search the index directly

## Agent-Friendly Features

1. **Dual Rendering**: HTML for humans, markdown for agents
2. **Structured Data**: OpenAPI spec, JSON-LD, machine-readable content
3. **Copy/Paste Commands**: Ready-to-use code examples
4. **Content Negotiation**: Automatic format detection
5. **Search API**: JSON search index for programmatic access

## Maintenance

### Updating Content

1. Edit markdown files in `content/` directory
2. Run `./deploy.sh` to rebuild and deploy
3. Changes are live immediately (static site)

### Adding New Pages

```bash
# Create new page
cd docs-site
hugo new content/section/page-name.md

# Edit the page
vim content/section/page-name.md

# Test locally
hugo server -D

# Deploy
./deploy.sh
```

### Monitoring

```bash
# Check nginx status
sudo systemctl status nginx

# Check site logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# Check Go middleware (if enabled)
sudo systemctl status darkpawns-docs
sudo journalctl -u darkpawns-docs -f
```

## Troubleshooting

### Site Not Loading
- Check nginx is running: `sudo systemctl status nginx`
- Check configuration: `sudo nginx -t`
- Check permissions: `ls -la /var/www/darkpawns/docs/`

### Content Negotiation Not Working
- Check Accept header is being sent
- Verify nginx configuration includes content negotiation rules
- Test with curl: `curl -H "Accept: text/markdown" http://localhost/`

### Search Not Working
- Check if `search-index.json` exists in public directory
- Verify JavaScript is loading (check browser console)
- Regenerate search index: `go run middleware.go -generate-index`

## Security Considerations

1. **SSL/TLS**: Always use HTTPS in production
2. **Rate Limiting**: Configured in nginx for API endpoints
3. **File Permissions**: Web server should have read-only access
4. **Content Security Policy**: Configured in nginx headers
5. **Regular Updates**: Keep Hugo, nginx, and system packages updated

## Performance

- Static site (fast, no database queries)
- Gzip compression enabled
- Browser caching for static assets
- CDN-ready (can be deployed to Cloudflare, Netlify, etc.)

## Backup

```bash
# Backup content
tar -czf darkpawns-docs-backup-$(date +%Y%m%d).tar.gz docs-site/

# Backup deployed site
tar -czf darkpawns-deployed-backup-$(date +%Y%m%d).tar.gz /var/www/darkpawns/docs/
```

## Support

- **Discord**: https://discord.gg/darkpawns
- **GitHub Issues**: https://github.com/zax0rz/darkpawns/issues
- **Email**: hello@labz0rz.com
EOF
    
    print_status "Deployment README generated at $SITE_DIR/DEPLOYMENT.md"
}

# Main execution
main() {
    print_status "Starting Dark Pawns documentation site deployment..."
    
    check_dependencies
    build_site
    deploy_files
    
    if command -v nginx &> /dev/null; then
        configure_nginx
        setup_ssl
    else
        print_warning "nginx not installed. Web server configuration skipped."
    fi
    
    if command -v go &> /dev/null; then
        create_systemd_service
    fi
    
    generate_readme
    
    print_status "Deployment complete!"
    print_status "Site URL: https://darkpawns.labz0rz.com/docs/"
    print_status "Markdown access: curl -H 'Accept: text/markdown' https://darkpawns.labz0rz.com/docs/"
}

# Run main function
main "$@"