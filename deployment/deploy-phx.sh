#!/usr/bin/env bash
# Deploy client/phx to the PHX production instance.
# Usage: ./deployment/deploy-phx.sh [--backend-only | --frontend-only]
#
# Prerequisites: AWS CLI configured (default profile works).

set -euo pipefail

INSTANCE_ID="i-0ea8b999ea9a7d908"
COMPOSE_DIR="/data/onyx/app/deployment/docker_compose"
COMPOSE_CMD="docker compose -f docker-compose.prod.yml -p onyx-stack"

BUILD_BACKEND=true
BUILD_FRONTEND=true

for arg in "$@"; do
  case $arg in
    --backend-only)  BUILD_FRONTEND=false ;;
    --frontend-only) BUILD_BACKEND=false ;;
  esac
done

# ── helpers ──────────────────────────────────────────────────────────────────

run_ssm() {
  local description="$1"
  local command="$2"
  local timeout="${3:-120}"

  echo ""
  echo "▶  $description"

  local cmd_id
  cmd_id=$(aws ssm send-command \
    --instance-ids "$INSTANCE_ID" \
    --document-name "AWS-RunShellScript" \
    --parameters "{\"commands\":[\"$command\"]}" \
    --timeout-seconds "$timeout" \
    --query 'Command.CommandId' \
    --output text)

  # Poll until done
  local status="InProgress"
  while [[ "$status" == "InProgress" || "$status" == "Pending" ]]; do
    sleep 8
    status=$(aws ssm get-command-invocation \
      --command-id "$cmd_id" \
      --instance-id "$INSTANCE_ID" \
      --query 'Status' \
      --output text 2>/dev/null || echo "Pending")
  done

  local stdout stderr
  stdout=$(aws ssm get-command-invocation \
    --command-id "$cmd_id" \
    --instance-id "$INSTANCE_ID" \
    --query 'StandardOutputContent' \
    --output text 2>/dev/null)
  stderr=$(aws ssm get-command-invocation \
    --command-id "$cmd_id" \
    --instance-id "$INSTANCE_ID" \
    --query 'StandardErrorContent' \
    --output text 2>/dev/null)

  if [[ "$status" != "Success" ]]; then
    echo "✗  FAILED (status: $status)"
    [[ -n "$stdout" ]] && echo "$stdout"
    [[ -n "$stderr" ]] && echo "$stderr" >&2
    exit 1
  fi

  [[ -n "$stdout" ]] && echo "$stdout"
}

# ── pre-flight ────────────────────────────────────────────────────────────────

echo "╔══════════════════════════════════════════╗"
echo "║  PHX Production Deploy                   ║"
echo "╚══════════════════════════════════════════╝"

# Verify SSM is reachable
ping_status=$(aws ssm describe-instance-information \
  --filters "Key=InstanceIds,Values=$INSTANCE_ID" \
  --query 'InstanceInformationList[0].PingStatus' \
  --output text 2>/dev/null || echo "Unknown")

if [[ "$ping_status" != "Online" ]]; then
  echo "✗  SSM agent is not online (status: $ping_status). Cannot deploy."
  exit 1
fi
echo "✓  SSM agent online"

# ── step 1: pull code ─────────────────────────────────────────────────────────

run_ssm "Pulling origin/client/phx" \
  "cd /data/onyx/app && git fetch origin && git reset --hard origin/client/phx && git clean -fd web/ && git log --oneline -2"

# ── step 2: build ─────────────────────────────────────────────────────────────

if $BUILD_BACKEND; then
  run_ssm "Building backend (api_server + background) — this takes ~5 min" \
    "cd $COMPOSE_DIR && $COMPOSE_CMD build api_server background > /tmp/build-backend.log 2>&1; echo EXIT=\$?; tail -3 /tmp/build-backend.log" \
    600
fi

if $BUILD_FRONTEND; then
  run_ssm "Building frontend (web_server) — this takes ~5 min" \
    "cd $COMPOSE_DIR && $COMPOSE_CMD build web_server > /tmp/build-web.log 2>&1; echo EXIT=\$?; tail -3 /tmp/build-web.log" \
    600
fi

# ── step 3: restart app containers ───────────────────────────────────────────

SERVICES=""
$BUILD_BACKEND  && SERVICES="$SERVICES api_server background"
$BUILD_FRONTEND && SERVICES="$SERVICES web_server"
SERVICES="${SERVICES# }"  # trim leading space

run_ssm "Restarting $SERVICES" \
  "cd $COMPOSE_DIR && $COMPOSE_CMD up -d --force-recreate --no-deps $SERVICES 2>&1"

# ── step 4: wait for api_server, then cycle nginx ────────────────────────────

run_ssm "Waiting for api_server health, then restarting nginx" \
  "until docker exec onyx-stack-api_server-1 curl -sf http://localhost:8080/health > /dev/null 2>&1; do sleep 5; done && echo API_HEALTHY && cd $COMPOSE_DIR && $COMPOSE_CMD stop nginx && $COMPOSE_CMD up -d nginx && sleep 10 && curl -sf http://localhost/ > /dev/null && echo SITE_UP" \
  300

# ── done ──────────────────────────────────────────────────────────────────────

echo ""
echo "✓  Deploy complete — https://phx.bemeaningful.ai"
