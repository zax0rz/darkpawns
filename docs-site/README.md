# Dark Pawns Documentation Site

A Hugo-based documentation website with dual rendering (HTML for humans, markdown for agents) for the Dark Pawns MUD resurrection project.

## Features

### 1. Dual Rendering (Content Negotiation)
- **HTML**: Beautiful, responsive web pages for human users
- **Markdown**: Clean markdown versions for AI agents via `Accept: text/markdown` header
- **JSON**: Structured data (OpenAPI spec, search index) for programmatic access

### 2. Agent-Friendly Documentation
- **Copy/Paste Commands**: Ready-to-use code examples in Python and JavaScript
- **WebSocket RPC Examples**: Typed methods with error handling
- **Structured Data**: OpenAPI 3.0 specification for API documentation
- **Search Functionality**: Full-text search for humans and agents

### 3. Complete Documentation Sections
- **Getting Started**: Installation and quick start guides
- **Game Documentation**: Commands, combat system, world guide
- **Agent Integration**: WebSocket protocol, example agents, memory systems
- **API Reference**: Complete WebSocket and REST API documentation
- **Development**: Contributing guide, architecture, testing, deployment

## Quick Start

### Local Development
```bash
# Clone the repository
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns/docs-site

# Start Hugo development server
hugo server -D

# Visit http://localhost:1313/
```

### Testing Content Negotiation
```bash
# Test with curl
curl http://localhost:1313/                         # HTML (default)
curl -H "Accept: text/markdown" http://localhost:1313/  # Markdown for agents
curl http://localhost:1313/api/openapi.json         # OpenAPI spec
curl http://localhost:1313/search-index.json        # Search index

# Run automated tests
python3 test_content_negotiation.py
```

### Deployment
```bash
# Build the site
hugo --minify

# Deploy using the deployment script
./deploy.sh
```

## Architecture

### Content Negotiation Flow
```
User/Agent Request
       │
       ▼ (Accept: text/html or text/markdown)
┌──────────────┐
│   nginx      │ ← Content negotiation rules
│   Reverse    │   in nginx.conf
│   Proxy      │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Hugo       │ ← Dual output formats
│   Static     │   (HTML + Markdown)
│   Site       │
└──────────────┘
```

### Directory Structure
```
docs-site/
├── content/                 # Markdown content
│   ├── _index.md           # Home page with agent-friendly examples
│   ├── getting-started/    # Installation and quick start
│   ├── agents/             # Agent integration guide
│   ├── api/                # API reference (includes openapi.json)
│   └── development/        # Development and contribution
├── themes/darkpawns-docs/  # Custom Hugo theme
│   ├── layouts/            # HTML templates with dual rendering
│   ├── assets/             # CSS and JavaScript
│   └── static/             # Static assets
├── middleware.go           # Go middleware for content negotiation
├── nginx.conf             # Production nginx configuration
├── deploy.sh              # Automated deployment script
├── test_content_negotiation.py # Test suite
└── public/                # Generated site (do not edit)
```

## Content Negotiation Implementation

### 1. Hugo Configuration (`hugo.toml`)
```toml
[outputs]
  home = ["HTML", "RSS", "JSON"]
  page = ["HTML", "Markdown"]

[outputFormats]
  [outputFormats.Markdown]
    mediaType = "text/markdown"
    baseName = "index"
    isPlainText = true
```

### 2. Template Support
- `layouts/_default/single.html` - HTML template
- `layouts/_default/single.markdown` - Markdown template
- Format selector in sidebar for easy switching

### 3. Web Server Configuration
- **nginx**: Content negotiation rules in `nginx.conf`
- **Go middleware**: Alternative implementation in `middleware.go`
- **Static serving**: Fast, CDN-ready deployment

## Agent Integration Examples

### Python Agent
```python
import websocket
import json

# Connect with content negotiation for docs
import requests
docs = requests.get(
    "https://darkpawns.labz0rz.com/docs/agents/protocol/",
    headers={"Accept": "text/markdown"}
).text

# Parse documentation and extract protocol info
# ... agent logic using the documentation ...
```

### Command Line Access
```bash
# Get documentation in markdown format
curl -H "Accept: text/markdown" https://darkpawns.labz0rz.com/docs/agents/protocol/ > protocol.md

# Get OpenAPI spec
curl https://darkpawns.labz0rz.com/docs/api/openapi.json > openapi.json

# Search the documentation
curl https://darkpawns.labz0rz.com/docs/search-index.json | jq '.[] | select(.title | contains("protocol"))'
```

## Search Functionality

### For Humans
- JavaScript-based search in sidebar
- Real-time results as you type
- Highlights matching terms

### For Agents
- JSON search index at `/search-index.json`
- Includes titles, descriptions, content snippets
- Can be downloaded and searched programmatically

### Search Index Structure
```json
[
  {
    "url": "/docs/agents/protocol/",
    "title": "WebSocket Protocol",
    "description": "Complete WebSocket protocol specification",
    "content": "Dark Pawns uses a WebSocket-based protocol...",
    "tags": ["api", "websocket", "agents"]
  }
]
```

## Deployment Options

### 1. Simple Static Hosting
```bash
# Build and copy files
hugo --minify
rsync -av public/ user@server:/var/www/darkpawns/docs/
```

### 2. Automated Deployment
```bash
# Uses the deploy.sh script
./deploy.sh
# - Builds site with minification
# - Generates search index
# - Configures nginx with SSL
# - Sets up systemd service for middleware
```

### 3. Docker Deployment
```bash
# Build Docker image
docker build -t darkpawns-docs .

# Run container
docker run -p 80:80 -p 443:443 darkpawns-docs
```

### 4. Cloud Platforms
- **Netlify/Vercel**: Connect GitHub repository
- **AWS S3 + CloudFront**: Static hosting with CDN
- **GitHub Pages**: Free hosting for open source

## Maintenance

### Adding New Content
```bash
# Create new page
hugo new content/section/page-name.md

# Add agent-friendly features
# - copy_paste_commands in frontmatter
# - api_examples for code samples
# - agent_friendly: true flag

# Test locally
hugo server -D

# Deploy
./deploy.sh
```

### Updating Search Index
The search index is automatically generated during build. To manually regenerate:
```bash
cd docs-site
go run middleware.go -generate-index
hugo --minify
```

### Monitoring
```bash
# Check nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# Check Hugo build
hugo --minify --verbose

# Test content negotiation
./test_content_negotiation.py
```

## Performance

- **Static Site**: No database, fast loading
- **Gzip Compression**: Enabled for all text content
- **Browser Caching**: Assets cached for 1 year
- **CDN Ready**: Can be deployed to any CDN
- **Lightweight**: Minimal JavaScript, optimized CSS

## Security

- **HTTPS Required**: SSL/TLS in production
- **Security Headers**: CSP, HSTS, XSS protection
- **Rate Limiting**: For API endpoints
- **Content Validation**: Markdown sanitization
- **Regular Updates**: Hugo and dependencies

## Testing

Run the test suite to verify all features:
```bash
# Start Hugo server
hugo server -D &

# Run tests
python3 test_content_negotiation.py

# Test specific features
python3 -c "
import requests
# Test markdown rendering
r = requests.get('http://localhost:1313/', headers={'Accept': 'text/markdown'})
print('Markdown test:', 'OK' if r.status_code == 200 else 'FAILED')
"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add or update documentation
4. Test with `./test_content_negotiation.py`
5. Submit a pull request

### Documentation Guidelines
- Use clear, concise language
- Include examples for all features
- Add `agent_friendly: true` to frontmatter for agent-relevant pages
- Include copy/paste commands where applicable
- Test both HTML and markdown rendering

## Support

- **Discord**: https://discord.gg/darkpawns
- **GitHub Issues**: https://github.com/zax0rz/darkpawns/issues
- **Documentation**: This site itself!

## License

MIT License - see [LICENSE](https://github.com/zax0rz/darkpawns/blob/main/LICENSE) file in the main repository.

---

*Part of the Dark Pawns MUD resurrection project. Originally created 1997-2010, resurrected with modern infrastructure and AI agent support.*