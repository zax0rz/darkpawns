# Dark Pawns Website — Deploy

## Architecture
- **Static files:** Hugo output in `public/` (rsync to VM 666)
- **Reverse proxy:** Caddy 2 (Docker) on VM 666 (192.168.1.125)
- **Content negotiation:** Post-build script generates `.md` files from content sources
- **Game server:** Go binary on VM 666, proxied via Caddy

## Deploy steps

```bash
# 1. Build Hugo site
cd website && hugo --minify

# 2. Generate markdown files for content negotiation
bash scripts/generate-markdown.sh

# 3. Sync to VM 666
rsync -avz --delete public/ root@192.168.1.125:/srv/darkpawns/

# 4. Caddy is running in Docker, restart if config changed
ssh root@192.168.1.125 "cd /opt/stacks/caddy && docker compose restart"
```

## DNS
- `darkpawns.labz0rz.com` → A record → 192.168.1.125 (Cloudflare, DNS-only)

## Caddy routes
- `/` → Static Hugo files
- `/ws` → WebSocket proxy → Go server (localhost:8080)
- `/api/*` → REST API proxy → Go server (localhost:8080)
- `/health` → Health check proxy → Go server (localhost:8080)
- `Accept: text/markdown` → serves `.md` files from content negotiation layer
