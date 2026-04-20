#!/usr/bin/env bash
# Local dev: Docker deps (optional) → alembic → web + API + model server (+ Celery if DEV_SKIP_BG=0).
# Requires: Docker, repo-root .venv (see CONTRIBUTING.md), and `npm install` in web/ and repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

SKIP_INFRA="${DEV_SKIP_INFRA:-0}"
SKIP_MIGRATE="${DEV_SKIP_MIGRATE:-0}"

if [[ ! -x "$ROOT/.venv/bin/python" ]]; then
  echo "error: $ROOT/.venv/bin/python not found. Create the venv from the repo root (see CONTRIBUTING.md Development Setup)." >&2
  exit 1
fi

PY="$ROOT/.venv/bin/python"
UVI="$ROOT/.venv/bin/uvicorn"

if [[ "$SKIP_INFRA" != "1" ]]; then
  echo "[dev-local] Starting infra (Postgres, OpenSearch, Redis, MinIO)…"
  docker compose -f deployment/docker_compose/docker-compose.yml \
    -f deployment/docker_compose/docker-compose.dev.yml \
    up -d index relational_db cache minio opensearch
else
  echo "[dev-local] DEV_SKIP_INFRA=1 — skipping docker compose"
fi

if [[ "$SKIP_MIGRATE" != "1" ]]; then
  echo "[dev-local] Running alembic upgrade head…"
  (cd "$ROOT/backend" && "$ROOT/.venv/bin/alembic" upgrade head)
else
  echo "[dev-local] DEV_SKIP_MIGRATE=1 — skipping migrations"
fi

export AUTH_TYPE="${AUTH_TYPE:-basic}"

CONC="${ROOT}/node_modules/.bin/concurrently"
if [[ ! -x "$CONC" ]]; then
  echo "error: concurrently not found. From repo root run: npm install" >&2
  exit 1
fi

SKIP_BG="${DEV_SKIP_BG:-1}"
if [[ "$SKIP_BG" == "1" ]]; then
  echo "[dev-local] Skipping Celery/beat (default). Set DEV_SKIP_BG=0 in .env for indexing/connectors background work."
  exec "$CONC" \
    --kill-others-on-fail false \
    -n web,api,model \
    -c cyan,magenta,yellow \
    "cd \"$ROOT/web\" && npm run dev" \
    "cd \"$ROOT/backend\" && env AUTH_TYPE=\"$AUTH_TYPE\" \"$UVI\" onyx.main:app --reload --port 8080" \
    "cd \"$ROOT/backend\" && \"$UVI\" model_server.main:app --reload --port 9000"
else
  echo "[dev-local] Running Celery/beat (DEV_SKIP_BG=0)."
  exec "$CONC" \
    --kill-others-on-fail false \
    -n web,api,model,bg \
    -c cyan,magenta,yellow,green \
    "cd \"$ROOT/web\" && npm run dev" \
    "cd \"$ROOT/backend\" && env AUTH_TYPE=\"$AUTH_TYPE\" \"$UVI\" onyx.main:app --reload --port 8080" \
    "cd \"$ROOT/backend\" && \"$UVI\" model_server.main:app --reload --port 9000" \
    "cd \"$ROOT/backend\" && \"$PY\" ./scripts/dev_run_background_jobs.py"
fi
