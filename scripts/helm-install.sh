#!/usr/bin/env bash
set -euo pipefail

IMAGE_SOURCE="tar"
IMAGE_DIR="image-tars"
VALUES_FILE="deploy/helm/k8s-ai-ops/values-local.yaml"
REGISTRY=""
TAG="local"
RELEASE="k8s-ai-ops"
NAMESPACE="k8s-ai-system"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image-source) IMAGE_SOURCE="$2"; shift 2 ;;
    --image-dir) IMAGE_DIR="$2"; shift 2 ;;
    --values) VALUES_FILE="$2"; shift 2 ;;
    --registry) REGISTRY="$2"; shift 2 ;;
    --tag) TAG="$2"; shift 2 ;;
    --release) RELEASE="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ "$IMAGE_SOURCE" == "tar" && ! -d "$IMAGE_DIR" ]]; then
  echo "image tar directory does not exist: $IMAGE_DIR" >&2
  exit 1
fi

helm upgrade --install "$RELEASE" deploy/helm/k8s-ai-ops \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --values "$VALUES_FILE" \
  --set images.source="$IMAGE_SOURCE" \
  --set images.registry="$REGISTRY" \
  --set images.tag="$TAG"

kubectl rollout status deployment/backend-api -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/mcp-server -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/frontend -n "$NAMESPACE" --timeout=180s

echo "frontend service: kubectl port-forward -n $NAMESPACE svc/frontend 8088:80"
echo "keycloak service: kubectl port-forward -n $NAMESPACE svc/keycloak 8089:8080"
