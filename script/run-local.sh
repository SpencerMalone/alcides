#!/usr/bin/env bash

set -euo pipefail

go build -o alcides ./cmd/alcides
echo "Built!"
source local.env
./alcides