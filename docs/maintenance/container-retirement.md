# Container deployment retirement — 2026-09-09

The maintained installation path is a native Go binary with PostgreSQL, as
documented in [Running Dark Pawns](../../DEPLOYMENT.md). The inherited container
setup did not have a verified build-to-login workflow, and maintaining a second
installation path is not a current project requirement.

## Removed from the working tree

- Five root image recipes: `Dockerfile`, `Dockerfile.ai-agent`, `Dockerfile.local`,
  `Dockerfile.prebuilt`, and `Dockerfile.privacy-filter`.
- Four root Compose files: the main, monitoring, moderation, and privacy stacks.
- `website/deploy/docker-compose.yml`, the obsolete Caddy container recipe.
- `deployment/deploy-local.sh`, `deployment/test-deployment.sh`, and
  `TEST_BUILD.sh`, which depended on the retired recipes.
- The web helper's Docker mode and its usage entry.
- Make targets that start, stop, build, or inspect the container stacks.
- CI's `build-and-push` job, registry settings, and package-write permission.
- Container installation instructions and requirements in maintained guides and
  the setup-wizard draft.

Kubernetes manifests and its deploy job were already absent. This pass removes
the remaining claims that they are supported. Removed files remain recoverable
from Git history, including revision `0505c87b7` before this cleanup.

## Retained

- CI's disposable PostgreSQL and Redis service containers, used only for tests.
- The native server, database migrations, moderation/privacy integrations, and
  `/metrics` endpoint. Gameplay behavior is unchanged (R1/R4).
- The optional Python privacy API, Prometheus/Grafana reference assets, and Caddy
  site configuration. These are not an automatically provisioned service stack.
- Historical briefs, research notes, and dated audits as evidence of past work.

No running container, remote registry package, Kubernetes cluster, VPS service,
production data, or private ops configuration was changed. The native installation
was separately [verified through a saved login](native-install-check.md).

## Future support

A container distribution can be reconsidered for a concrete user need. It would
need its own automated build, startup, character login, and persistence/restart
checks before becoming an advertised installation option.
