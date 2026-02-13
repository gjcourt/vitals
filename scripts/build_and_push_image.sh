#!/bin/bash
set -e

# specific registry and image name
REGISTRY="ghcr.io"
USER="gjcourt"
PROJECT="biometrics"
DATE=$(date +%Y-%m-%d)

IMAGE="$REGISTRY/$USER/$PROJECT:$DATE"

echo "Building image: $IMAGE"
docker build -t "$IMAGE" .

echo "Pushing image: $IMAGE"
docker push "$IMAGE"

echo "Done! Image pushed to $IMAGE"
