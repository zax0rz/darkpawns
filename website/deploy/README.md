# Dark Pawns Website — Deploy

## Architecture
- **Static files:** Hugo output in `public/` (rsync to `frankendell`)
- **Reverse proxy:** Caddy 2 (Docker) on `frankendell` (192.168.1.15)
- **Content negotiation:** Native Hugo Markdown content formats, served via Caddy Accept rules
- **Game server:** Go binary on `frankendell`, proxied via Caddy

## Deploy steps

To build and deploy the website to live:

```bash
# 1. Build Hugo site with minification
cd website && hugo --minify

# 2. Sync files directly to the live server
rsync -avz --delete public/ root@192.168.1.15:/opt/darkpawns/hugo-site/
```

*Note: Caddy bind-mounts this directory (`/srv/hugo` inside the container), so the updates are live instantly with **no container or Caddy restart needed**.*

## DNS
- `darkpawns.labz0rz.com` → A record → Cloudflare DNS → proxied to `frankendell` (192.168.1.15)

## Caddy routes
- `/` → Static Hugo files (served from bind-mount `/opt/darkpawns/hugo-site/`)
- `/ws` → WebSocket proxy → Go server (localhost:4350)
- `/api/*` → REST API proxy → Go server (localhost:4350)
- `/health` → Health check proxy → Go server (localhost:4350)
- `Accept: text/markdown` → serves `.md` files natively from Hugo output format routes
