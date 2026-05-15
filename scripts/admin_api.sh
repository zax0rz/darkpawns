#!/usr/bin/env bash
# Admin API helper — used by Reek and Daeron to self-report findings/status
#
# Usage:
#   ./scripts/admin_api.sh status <agent_id> <status> [model]
#   ./scripts/admin_api.sh finding <source> <severity> <title> <file> <line> <description>
#   ./scripts/admin_api.sh update-finding <id> <status> [linear_issue_id]
#   ./scripts/admin_api.sh triage <date> <confirmed> <rejected> <pending> <summary>
#
# Requires DP_ADMIN_TOKEN env var (builder-scoped JWT)
# Requires DP_ADMIN_URL env var (default: http://192.168.1.125:4350)

set -euo pipefail

DP_ADMIN_URL="${DP_ADMIN_URL:-http://192.168.1.125:4350}"

if [[ -z "${DP_ADMIN_TOKEN:-}" ]]; then
  echo "ERROR: DP_ADMIN_TOKEN not set" >&2
  exit 1
fi

CMD="${1:-}"
shift || true

case "$CMD" in
  status)
    AGENT_ID="$1"
    STATUS="$2"
    MODEL="${3:-}"
    if [[ -z "$AGENT_ID" || -z "$STATUS" ]]; then
      echo "Usage: admin_api.sh status <agent_id> <status> [model]" >&2
      exit 1
    fi
    PAYLOAD="{\"agent_id\":\"$AGENT_ID\",\"status\":\"$STATUS\""
    if [[ -n "$MODEL" ]]; then
      PAYLOAD="$PAYLOAD,\"model\":\"$MODEL\""
    fi
    PAYLOAD="$PAYLOAD}"
    curl -sf -X POST "$DP_ADMIN_URL/admin/agents/status" \
      -H "Authorization: Bearer $DP_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$PAYLOAD" 2>/dev/null || echo "WARN: admin API unreachable" >&2
    ;;

  finding)
    SOURCE="$1"
    SEVERITY="$2"
    TITLE="$3"
    FILE="$4"
    LINE="$5"
    DESC="$6"
    if [[ -z "$SOURCE" || -z "$SEVERITY" || -z "$TITLE" ]]; then
      echo "Usage: admin_api.sh finding <source> <severity> <title> <file> <line> <desc>" >&2
      exit 1
    fi
    # Escape description for JSON (replace " with \")
    ESCAPED_DESC="${DESC//\"/\\\"}"
    ESCAPED_TITLE="${TITLE//\"/\\\"}"
    curl -sf -X POST "$DP_ADMIN_URL/admin/findings" \
      -H "Authorization: Bearer $DP_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"source\":\"$SOURCE\",\"severity\":\"$SEVERITY\",\"title\":\"$ESCAPED_TITLE\",\"file\":\"$FILE\",\"line\":$LINE,\"description\":\"$ESCAPED_DESC\"}" 2>/dev/null \
      || echo "WARN: admin API unreachable" >&2
    ;;

  update-finding)
    ID="$1"
    STATUS="${2:-}"
    LINEAR_ID="${3:-}"
    if [[ -z "$ID" ]]; then
      echo "Usage: admin_api.sh update-finding <id> [status] [linear_issue_id]" >&2
      exit 1
    fi
    PAYLOAD="{"
    if [[ -n "$STATUS" ]]; then
      PAYLOAD="$PAYLOAD\"status\":\"$STATUS\""
    fi
    if [[ -n "$LINEAR_ID" ]]; then
      [[ -n "$STATUS" ]] && PAYLOAD="$PAYLOAD,"
      PAYLOAD="$PAYLOAD\"linear_issue_id\":\"$LINEAR_ID\""
    fi
    PAYLOAD="$PAYLOAD}"
    curl -sf -X PUT "$DP_ADMIN_URL/admin/findings/$ID" \
      -H "Authorization: Bearer $DP_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$PAYLOAD" 2>/dev/null \
      || echo "WARN: admin API unreachable" >&2
    ;;

  triage)
    DATE="$1"
    CONFIRMED="$2"
    REJECTED="$3"
    PENDING="$4"
    SUMMARY="$5"
    if [[ -z "$DATE" ]]; then
      echo "Usage: admin_api.sh triage <date> <confirmed> <rejected> <pending> <summary>" >&2
      exit 1
    fi
    ESCAPED_SUMMARY="${SUMMARY//\"/\\\"}"
    curl -sf -X POST "$DP_ADMIN_URL/admin/triage/summaries" \
      -H "Authorization: Bearer $DP_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"date\":\"$DATE\",\"confirmed\":$CONFIRMED,\"rejected\":$REJECTED,\"pending\":$PENDING,\"summary\":\"$ESCAPED_SUMMARY\"}" 2>/dev/null \
      || echo "WARN: admin API unreachable" >&2
    ;;

  *)
    echo "Usage: admin_api.sh {status|finding|update-finding|triage} ..." >&2
    exit 1
    ;;
esac
