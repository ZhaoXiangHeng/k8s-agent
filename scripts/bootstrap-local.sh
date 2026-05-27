#!/usr/bin/env bash
set -euo pipefail

IMAGE_SOURCE="tar"
IMAGE_DIR="image-tars"
CLUSTER_NAME="k8s-ai"
VALUES_FILE="deploy/helm/k8s-ai-ops/values-local.yaml"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image-source) IMAGE_SOURCE="$2"; shift 2 ;;
    --image-dir) IMAGE_DIR="$2"; shift 2 ;;
    --cluster-name) CLUSTER_NAME="$2"; shift 2 ;;
    --values) VALUES_FILE="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

for bin in docker kind kubectl helm; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "missing required command: $bin" >&2
    exit 1
  fi
done

if ! kind get clusters | grep -qx "$CLUSTER_NAME"; then
  kind create cluster --name "$CLUSTER_NAME"
fi

kubectl create namespace dev --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace test --dry-run=client -o yaml | kubectl apply -f -

if [[ "$IMAGE_SOURCE" == "tar" ]]; then
  for image_archive in "$IMAGE_DIR"/*.tar; do
    [[ -e "$image_archive" ]] || { echo "no image archives found in $IMAGE_DIR" >&2; exit 1; }
    kind load image-archive "$image_archive" --name "$CLUSTER_NAME"
  done
fi

"$(dirname "$0")/helm-install.sh" --image-source "$IMAGE_SOURCE" --values "$VALUES_FILE"
