#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

GOPROXY_VALUE="${GOPROXY:-https://goproxy.cn,direct}"
GOSUMDB_VALUE="${GOSUMDB:-sum.golang.google.cn}"
NO_CACHE_VALUE="${NO_CACHE:-1}"

BUILD_ARGS=(
  --build-arg "GOPROXY=${GOPROXY_VALUE}"
  --build-arg "GOSUMDB=${GOSUMDB_VALUE}"
)

if [[ "$NO_CACHE_VALUE" == "1" ]]; then
  CACHE_ARGS=(--no-cache)
else
  CACHE_ARGS=()
fi

echo "==> Building image with GOPROXY=${GOPROXY_VALUE}, GOSUMDB=${GOSUMDB_VALUE}"
docker compose build "${CACHE_ARGS[@]}" "${BUILD_ARGS[@]}"

echo "==> Starting service"
docker compose up -d

echo "==> Service status"
docker compose ps
