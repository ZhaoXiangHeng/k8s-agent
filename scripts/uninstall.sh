#!/usr/bin/env bash
set -euo pipefail

RELEASE="k8s-ai-ops"
NAMESPACE="k8s-ai-system"
DELETE_DATA="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release) RELEASE="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    --delete-data) DELETE_DATA="true"; shift ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

helm uninstall "$RELEASE" --namespace "$NAMESPACE"

if [[ "$DELETE_DATA" == "true" ]]; then
  kubectl delete pvc -n "$NAMESPACE" --all
fi
