#!/usr/bin/env bash
set -euo pipefail

TAG="local"
OUTPUT_DIR="image-tars"
SAVE_TARS="true"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) TAG="$2"; shift 2 ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --save-tars) SAVE_TARS="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

docker build -t "k8s-ai-backend:$TAG" backend
docker build -f agent-server/Dockerfile -t "k8s-ai-agent-server:$TAG" .
docker build -t "k8s-ai-mcp-server:$TAG" mcp-server
docker build -t "k8s-ai-frontend:$TAG" frontend

if [[ "$SAVE_TARS" == "true" ]]; then
  mkdir -p "$OUTPUT_DIR"
  docker save "k8s-ai-backend:$TAG" -o "$OUTPUT_DIR/backend-api-amd64.tar"
  docker save "k8s-ai-agent-server:$TAG" -o "$OUTPUT_DIR/agent-server-amd64.tar"
  docker save "k8s-ai-mcp-server:$TAG" -o "$OUTPUT_DIR/mcp-server-amd64.tar"
  docker save "k8s-ai-frontend:$TAG" -o "$OUTPUT_DIR/frontend-amd64.tar"
fi
