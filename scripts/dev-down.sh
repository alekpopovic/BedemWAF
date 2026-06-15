#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/../deployments"
docker compose down
