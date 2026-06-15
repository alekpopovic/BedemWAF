#!/usr/bin/env sh
set -eu

root_dir="$(dirname "$0")/.."

export GOCACHE="${GOCACHE:-/tmp/bedemwaf-go-cache}"

(cd "$root_dir/services/gateway" && go test ./...)
(cd "$root_dir/services/control-api" && go test ./...)
(cd "$root_dir/services/worker" && go test ./...)
