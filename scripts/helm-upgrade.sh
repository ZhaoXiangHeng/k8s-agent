#!/usr/bin/env bash
set -euo pipefail

IMAGE_SOURCE="registry"
VALUES_FILE="deploy/helm/k8s-ai-ops/values-prod-example.yaml"
REGISTRY=""
TAG="local"
RELEASE="k8s-ai-ops"
NAMESPACE="k8s-ai-system"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image-source) IMAGE_SOURCE="$2"; shift 2 ;;
    --values) VALUES_FILE="$2"; shift 2 ;;
    --registry) REGISTRY="$2"; shift 2 ;;
    --tag) TAG="$2"; shift 2 ;;
    --release) RELEASE="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

helm upgrade "$RELEASE" deploy/helm/k8s-ai-ops \
  --namespace "$NAMESPACE" \
  --values "$VALUES_FILE" \
  --set images.source="$IMAGE_SOURCE" \
  --set images.registry="$REGISTRY" \
  --set images.tag="$TAG"

kubectl rollout status deployment/backend-api -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/mcp-server -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/frontend -n "$NAMESPACE" --timeout=180s
