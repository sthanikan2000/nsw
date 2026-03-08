#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.docker}"

RUN_IDP=true
RUN_POSTGRES=true
RUN_MIGRATIONS=true
RUN_BUILD=true
STOP_ONLY=false

NETWORK_NAME="nsw-dev-net"
POSTGRES_CONTAINER="nsw-postgres"
BACKEND_CONTAINER="nsw-backend"
OGA_NPQS_CONTAINER="nsw-oga-npqs"
OGA_FCAU_CONTAINER="nsw-oga-fcau"
OGA_IRD_CONTAINER="nsw-oga-ird"
TRADER_PORTAL_CONTAINER="nsw-trader-portal"
OGA_PORTAL_NPQS_CONTAINER="nsw-oga-app-npqs"
OGA_PORTAL_FCAU_CONTAINER="nsw-oga-app-fcau"
OGA_PORTAL_IRD_CONTAINER="nsw-oga-app-ird"

usage() {
  cat <<EOF
Usage: ./start-docker.sh [options]

Options:
  --env-file=/path/to/.env   Use a custom environment file (default: ./.env.docker)
  --skip-idp                 Do not start IDP via docker compose
  --skip-postgres            Do not start PostgreSQL container
  --skip-migrations          Do not apply backend SQL migration into PostgreSQL
  --skip-build               Do not build images before running containers
  --stop                     Stop and remove NSW docker containers, then exit

Examples:
  ./start-docker.sh
  ./start-docker.sh --skip-build
  ./start-docker.sh --stop
EOF
}

for arg in "$@"; do
  case "$arg" in
    --env-file=*)
      ENV_FILE="${arg#*=}"
      ;;
    --skip-idp)
      RUN_IDP=false
      ;;
    --skip-postgres)
      RUN_POSTGRES=false
      ;;
    --skip-migrations)
      RUN_MIGRATIONS=false
      ;;
    --skip-build)
      RUN_BUILD=false
      ;;
    --stop)
      STOP_ONLY=true
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg"
      usage
      exit 1
      ;;
  esac
done

for cmd in docker; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd is required but was not found in PATH"
    exit 1
  fi
done

if [[ "$RUN_IDP" == "true" ]]; then
  if [[ ! -d "$ROOT_DIR/idp" ]]; then
    echo "IDP directory not found at $ROOT_DIR/idp"
    exit 1
  fi
fi

if [[ "$STOP_ONLY" == "true" ]]; then
  echo "Stopping NSW docker containers..."
  docker rm -f \
    "$BACKEND_CONTAINER" \
    "$OGA_NPQS_CONTAINER" \
    "$OGA_FCAU_CONTAINER" \
    "$OGA_IRD_CONTAINER" \
    "$TRADER_PORTAL_CONTAINER" \
    "$OGA_PORTAL_NPQS_CONTAINER" \
    "$OGA_PORTAL_FCAU_CONTAINER" \
    "$OGA_PORTAL_IRD_CONTAINER" \
    "$POSTGRES_CONTAINER" \
    >/dev/null 2>&1 || true

  if [[ "$RUN_IDP" == "true" ]]; then
    echo "Stopping IDP compose stack..."
    (
      cd "$ROOT_DIR/idp"
      docker compose down
    )
  fi

  echo "Done."
  exit 0
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Env file not found: $ENV_FILE"
  echo "Create one from: cp $ROOT_DIR/.env.example $ROOT_DIR/.env"
  exit 1
fi

if [[ "$ENV_FILE" != /* ]]; then
  ENV_FILE="$ROOT_DIR/$ENV_FILE"
fi

set -a
source "$ENV_FILE"
set +a

IDP_PORT="${IDP_PORT:-8090}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
TRADER_APP_PORT="${TRADER_APP_PORT:-5173}"
OGA_NPQS_PORT="${OGA_NPQS_PORT:-8081}"
OGA_FCAU_PORT="${OGA_FCAU_PORT:-8082}"
OGA_IRD_PORT="${OGA_IRD_PORT:-8083}"
OGA_APP_NPQS_PORT="${OGA_APP_NPQS_PORT:-5174}"
OGA_APP_FCAU_PORT="${OGA_APP_FCAU_PORT:-5175}"
OGA_APP_IRD_PORT="${OGA_APP_IRD_PORT:-5176}"

DB_PORT_HOST="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-nsw_db}"
DB_USERNAME="${DB_USERNAME:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-changeme}"
DB_SSLMODE="${DB_SSLMODE:-disable}"
DB_HOST="${DB_HOST:-localhost}"
BACKEND_DB_PORT="$DB_PORT_HOST"

SERVER_DEBUG="${SERVER_DEBUG:-true}"
SERVER_LOG_LEVEL="${SERVER_LOG_LEVEL:-info}"
CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:3000,http://localhost:5173,http://localhost:5174,http://localhost:5175,http://localhost:5176}"
SERVICE_URL="${SERVICE_URL:-${BACKEND_SERVICE_URL:-http://${BACKEND_CONTAINER}:${BACKEND_PORT}}}"

AUTH_ISSUER="${AUTH_ISSUER:-https://localhost:${IDP_PORT}}"
AUTH_JWKS_URL="${AUTH_JWKS_URL:-https://localhost:${IDP_PORT}/oauth2/jwks}"
AUTH_CLIENT_ID="${AUTH_CLIENT_ID:-TRADER_PORTAL_APP}"
AUTH_AUDIENCE="${AUTH_AUDIENCE:-TRADER_PORTAL_APP}"
AUTH_JWKS_INSECURE_SKIP_VERIFY="${AUTH_JWKS_INSECURE_SKIP_VERIFY:-true}"

IDP_PUBLIC_URL="${IDP_PUBLIC_URL:-https://localhost:${IDP_PORT}}"
TRADER_IDP_CLIENT_ID="${TRADER_IDP_CLIENT_ID:-TRADER_PORTAL_APP}"
NPQS_IDP_CLIENT_ID="${NPQS_IDP_CLIENT_ID:-OGA_PORTAL_APP_NPQS}"
FCAU_IDP_CLIENT_ID="${FCAU_IDP_CLIENT_ID:-OGA_PORTAL_APP_FCAU}"
IRD_IDP_CLIENT_ID="${IRD_IDP_CLIENT_ID:-OGA_PORTAL_APP_IRD}"
IDP_SCOPES="${IDP_SCOPES:-openid,profile,email}"
IDP_PLATFORM="${IDP_PLATFORM:-AsgardeoV2}"
SHOW_AUTOFILL_BUTTON="${SHOW_AUTOFILL_BUTTON:-true}"

OGA_DEFAULT_FORM_ID="${OGA_DEFAULT_FORM_ID:-default}"
OGA_ALLOWED_ORIGINS="${OGA_ALLOWED_ORIGINS:-*}"
OGA_APP_NPQS_INSTANCE_CONFIG="${OGA_APP_NPQS_INSTANCE_CONFIG:-npqs}"
OGA_APP_FCAU_INSTANCE_CONFIG="${OGA_APP_FCAU_INSTANCE_CONFIG:-fcau}"
OGA_APP_IRD_INSTANCE_CONFIG="${OGA_APP_IRD_INSTANCE_CONFIG:-ird}"

echo "Preparing Docker network..."
docker network create "$NETWORK_NAME" >/dev/null 2>&1 || true

if [[ "$RUN_IDP" == "true" ]]; then
  echo "Starting IDP..."
  (
    cd "$ROOT_DIR/idp"
    docker compose up -d
  )
fi

if [[ "$RUN_BUILD" == "true" ]]; then
  echo "Building images..."
  docker build -f "$ROOT_DIR/backend/Dockerfile" -t nsw-backend:local "$ROOT_DIR/backend"
  docker build -f "$ROOT_DIR/oga/Dockerfile" -t nsw-oga-backend:local "$ROOT_DIR/oga"
  docker build -f "$ROOT_DIR/portals/apps/trader-app/Dockerfile" -t nsw-trader-portal:local "$ROOT_DIR/portals"
  docker build -f "$ROOT_DIR/portals/apps/oga-app/Dockerfile" -t nsw-oga-portal:local "$ROOT_DIR/portals"
fi

if [[ "$RUN_POSTGRES" == "true" ]]; then
  echo "Starting PostgreSQL..."
  docker rm -f "$POSTGRES_CONTAINER" >/dev/null 2>&1 || true
  docker run -d --name "$POSTGRES_CONTAINER" \
    --network "$NETWORK_NAME" \
    -e POSTGRES_PASSWORD="$DB_PASSWORD" \
    -e POSTGRES_DB="$DB_NAME" \
    -p "$DB_PORT_HOST:5432" \
    postgres:16-alpine >/dev/null

  if [[ "$RUN_MIGRATIONS" == "true" ]]; then
    echo "Applying backend migrations via run.sh..."
    POSTGRES_READY=false
    for attempt in {1..30}; do
      if PGPASSWORD="$DB_PASSWORD" psql \
        -h localhost \
        -p "$DB_PORT_HOST" \
        -U "$DB_USERNAME" \
        -d "$DB_NAME" \
        -c 'SELECT 1' \
        >/dev/null 2>&1; then
        POSTGRES_READY=true
        break
      fi
      sleep 1
    done

    if [[ "$POSTGRES_READY" != "true" ]]; then
      echo "PostgreSQL did not become ready in time on localhost:$DB_PORT_HOST"
      exit 1
    fi

    (
      cd "$ROOT_DIR/backend/internal/database/migrations"
      ENV_FILE="$ENV_FILE" \
      DB_PORT="$DB_PORT_HOST" \
      MIGRATION_DB_HOST="localhost" \
      NPQS_OGA_SUBMISSION_URL="http://${OGA_NPQS_CONTAINER}:${OGA_NPQS_PORT}/api/oga/inject" \
      FCAU_OGA_SUBMISSION_URL="http://${OGA_FCAU_CONTAINER}:${OGA_FCAU_PORT}/api/oga/inject" \
      PRECONSIGNMENT_OGA_SUBMISSION_URL="http://${OGA_IRD_CONTAINER}:${OGA_IRD_PORT}/api/oga/inject" \
      bash ./run.sh
    )
  fi

  DB_HOST="$POSTGRES_CONTAINER"
  BACKEND_DB_PORT="5432"
elif [[ "$RUN_MIGRATIONS" == "true" ]]; then
  echo "Warning: --skip-postgres was used; skipping migrations because no managed postgres container is running."
fi

echo "Starting NSW containers..."
docker rm -f \
  "$BACKEND_CONTAINER" \
  "$OGA_NPQS_CONTAINER" \
  "$OGA_FCAU_CONTAINER" \
  "$OGA_IRD_CONTAINER" \
  "$TRADER_PORTAL_CONTAINER" \
  "$OGA_PORTAL_NPQS_CONTAINER" \
  "$OGA_PORTAL_FCAU_CONTAINER" \
  "$OGA_PORTAL_IRD_CONTAINER" \
  >/dev/null 2>&1 || true

docker run -d --name "$BACKEND_CONTAINER" \
  --network "$NETWORK_NAME" \
  --network-alias backend \
  -p "$BACKEND_PORT:8080" \
  -e DB_HOST="$DB_HOST" \
  -e DB_PORT="$BACKEND_DB_PORT" \
  -e DB_USERNAME="$DB_USERNAME" \
  -e DB_PASSWORD="$DB_PASSWORD" \
  -e DB_NAME="$DB_NAME" \
  -e DB_SSLMODE="$DB_SSLMODE" \
  -e SERVER_PORT="8080" \
  -e SERVICE_URL="$SERVICE_URL" \
  -e SERVER_DEBUG="$SERVER_DEBUG" \
  -e SERVER_LOG_LEVEL="$SERVER_LOG_LEVEL" \
  -e CORS_ALLOWED_ORIGINS="$CORS_ALLOWED_ORIGINS" \
  -e AUTH_JWKS_URL="$AUTH_JWKS_URL" \
  -e AUTH_ISSUER="$AUTH_ISSUER" \
  -e AUTH_CLIENT_ID="$AUTH_CLIENT_ID" \
  -e AUTH_AUDIENCE="$AUTH_AUDIENCE" \
  -e AUTH_JWKS_INSECURE_SKIP_VERIFY="$AUTH_JWKS_INSECURE_SKIP_VERIFY" \
  nsw-backend:local >/dev/null

docker run -d --name "$OGA_NPQS_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_NPQS_PORT:$OGA_NPQS_PORT" \
  -v nsw-oga-npqs-data:/data \
  -e OGA_PORT="$OGA_NPQS_PORT" \
  -e OGA_DEFAULT_FORM_ID="$OGA_DEFAULT_FORM_ID" \
  -e OGA_ALLOWED_ORIGINS="$OGA_ALLOWED_ORIGINS" \
  nsw-oga-backend:local >/dev/null

docker run -d --name "$OGA_FCAU_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_FCAU_PORT:$OGA_FCAU_PORT" \
  -v nsw-oga-fcau-data:/data \
  -e OGA_PORT="$OGA_FCAU_PORT" \
  -e OGA_DEFAULT_FORM_ID="$OGA_DEFAULT_FORM_ID" \
  -e OGA_ALLOWED_ORIGINS="$OGA_ALLOWED_ORIGINS" \
  nsw-oga-backend:local >/dev/null

docker run -d --name "$OGA_IRD_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_IRD_PORT:$OGA_IRD_PORT" \
  -v nsw-oga-ird-data:/data \
  -e OGA_PORT="$OGA_IRD_PORT" \
  -e OGA_DEFAULT_FORM_ID="$OGA_DEFAULT_FORM_ID" \
  -e OGA_ALLOWED_ORIGINS="$OGA_ALLOWED_ORIGINS" \
  nsw-oga-backend:local >/dev/null

docker run -d --name "$TRADER_PORTAL_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$TRADER_APP_PORT:80" \
  -e VITE_API_BASE_URL="http://localhost:${BACKEND_PORT}/api/v1" \
  -e VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  -e VITE_IDP_CLIENT_ID="$TRADER_IDP_CLIENT_ID" \
  -e VITE_APP_URL="http://localhost:${TRADER_APP_PORT}" \
  -e VITE_IDP_SCOPES="$IDP_SCOPES" \
  -e VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  -e VITE_SHOW_AUTOFILL_BUTTON="$SHOW_AUTOFILL_BUTTON" \
  nsw-trader-portal:local >/dev/null

docker run -d --name "$OGA_PORTAL_NPQS_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_APP_NPQS_PORT:80" \
  -e VITE_INSTANCE_CONFIG="$OGA_APP_NPQS_INSTANCE_CONFIG" \
  -e VITE_API_BASE_URL="http://localhost:${OGA_NPQS_PORT}" \
  -e VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  -e VITE_IDP_CLIENT_ID="$NPQS_IDP_CLIENT_ID" \
  -e VITE_APP_URL="http://localhost:${OGA_APP_NPQS_PORT}" \
  -e VITE_IDP_SCOPES="$IDP_SCOPES" \
  -e VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  nsw-oga-portal:local >/dev/null

docker run -d --name "$OGA_PORTAL_FCAU_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_APP_FCAU_PORT:80" \
  -e VITE_INSTANCE_CONFIG="$OGA_APP_FCAU_INSTANCE_CONFIG" \
  -e VITE_API_BASE_URL="http://localhost:${OGA_FCAU_PORT}" \
  -e VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  -e VITE_IDP_CLIENT_ID="$FCAU_IDP_CLIENT_ID" \
  -e VITE_APP_URL="http://localhost:${OGA_APP_FCAU_PORT}" \
  -e VITE_IDP_SCOPES="$IDP_SCOPES" \
  -e VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  nsw-oga-portal:local >/dev/null

docker run -d --name "$OGA_PORTAL_IRD_CONTAINER" \
  --network "$NETWORK_NAME" \
  -p "$OGA_APP_IRD_PORT:80" \
  -e VITE_INSTANCE_CONFIG="$OGA_APP_IRD_INSTANCE_CONFIG" \
  -e VITE_API_BASE_URL="http://localhost:${OGA_IRD_PORT}" \
  -e VITE_IDP_BASE_URL="$IDP_PUBLIC_URL" \
  -e VITE_IDP_CLIENT_ID="$IRD_IDP_CLIENT_ID" \
  -e VITE_APP_URL="http://localhost:${OGA_APP_IRD_PORT}" \
  -e VITE_IDP_SCOPES="$IDP_SCOPES" \
  -e VITE_IDP_PLATFORM="$IDP_PLATFORM" \
  nsw-oga-portal:local >/dev/null

cat <<EOF

Started Docker services:
  - backend       -> http://localhost:${BACKEND_PORT}
  - trader-app    -> http://localhost:${TRADER_APP_PORT}
  - oga-npqs      -> http://localhost:${OGA_NPQS_PORT}
  - oga-fcau      -> http://localhost:${OGA_FCAU_PORT}
  - oga-ird       -> http://localhost:${OGA_IRD_PORT}
  - oga-app-npqs  -> http://localhost:${OGA_APP_NPQS_PORT}
  - oga-app-fcau  -> http://localhost:${OGA_APP_FCAU_PORT}
  - oga-app-ird   -> http://localhost:${OGA_APP_IRD_PORT}

Supporting services:
  - postgres      -> localhost:${DB_PORT_HOST} (container: ${POSTGRES_CONTAINER}, if enabled)
  - idp           -> https://localhost:${IDP_PORT} (if enabled)

Useful commands:
  - View logs: docker logs -f ${BACKEND_CONTAINER}
  - Stop stack: ./start-docker.sh --stop
EOF
