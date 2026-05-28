#!/usr/bin/env bash
set -euo pipefail

TAG="local"
SERVICES="all"
BUILD_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) TAG="$2"; shift 2 ;;
    --services) SERVICES="$2"; shift 2 ;;
    --build-dir) BUILD_DIR="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$BUILD_DIR" ]]; then
  echo "ERROR: --build-dir is required" >&2
  exit 1
fi

cd "$BUILD_DIR"
echo "[build-on-server] Build dir: $BUILD_DIR"
echo "[build-on-server] Tag: $TAG"
echo "[build-on-server] Services: $SERVICES"

build_service() {
  local name="$1"
  local dockerfile="$2"
  local image="k8s-ai-${name}:${TAG}"
  echo "[build-on-server] === Building ${name} ==="
  docker build -f "$dockerfile" -t "$image" --build-arg LDFLAGS="-s -w" "$BUILD_DIR"
  echo "[build-on-server] Saving ${name} ..."
  docker save "$image" | ctr -n k8s.io images import -
  echo "[build-on-server] ${name} done"
}

if [[ "$SERVICES" == "all" || "$SERVICES" == *"mcp-server"* ]]; then
  build_service "mcp-server" "mcp-server/Dockerfile"
fi

if [[ "$SERVICES" == "all" || "$SERVICES" == *"agent-server"* ]]; then
  build_service "agent-server" "agent-server/Dockerfile"
fi

if [[ "$SERVICES" == "all" || "$SERVICES" == *"backend"* ]]; then
  build_service "backend" "backend/Dockerfile"
fi

if [[ "$SERVICES" == "all" || "$SERVICES" == *"frontend"* ]]; then
  echo "[build-on-server] === Building frontend ==="
  docker build -f "frontend/Dockerfile" -t "k8s-ai-frontend:${TAG}" "$BUILD_DIR"
  echo "[build-on-server] Saving frontend ..."
  docker save "k8s-ai-frontend:${TAG}" | ctr -n k8s.io images import -
  echo "[build-on-server] frontend done"
fi

echo "[build-on-server] === All builds complete ==="
ctr -n k8s.io images list | grep k8s-ai
