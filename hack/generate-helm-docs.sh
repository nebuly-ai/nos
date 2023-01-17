#!/bin/bash

HELM_CHARTS_DOCS_BASE_PATH="docs/en/docs/helm-charts"

for dir in helm-charts/*; do
  # If dir is not a directory then continue
  [[ -d "$dir" ]] || continue
  dir=$(basename "$dir")

  # Create dir under DOCS_BASE_PATH if not already exists
  if [ ! -d "$HELM_CHARTS_DOCS_BASE_PATH/$dir" ]; then
    mkdir -p "$HELM_CHARTS_DOCS_BASE_PATH/$dir"
  fi

  # Copy README.md to the docs directory
  echo "Copying $dir/README.md to $HELM_CHARTS_DOCS_BASE_PATH/$dir"
  cp "helm-charts/$dir/README.md" "$HELM_CHARTS_DOCS_BASE_PATH/$dir/README.md"
done