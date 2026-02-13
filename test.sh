#!/bin/sh
# Replace with your username and PAT
USER_NAME="gjcourt"
IMAGE="golinks"
GH_TOKEN=$GHCR_TOKEN

# Get an access token
TOKEN=$(curl -s "https://ghcr.io/token?service=ghcr.io&scope=repository:${USER_NAME}/${IMAGE}:pull" \
  -u "${USER_NAME}:${GH_TOKEN}" | jq -r '.token')
echo "Token generated $TOKEN"

echo "Listing tags..."
curl -H "Authorization: Bearer $TOKEN" \
  "https://ghcr.io/v2/${USER_NAME}/${IMAGE}/tags/list"   
