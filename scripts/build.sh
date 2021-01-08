#!/bin/bash

set -euo pipefail

# Disable CGO by default
export CGO_ENABLED=${CGO_ENABLED:-0}

BASENAME=$(basename $(pwd))

arch=(amd64)
os=(darwin linux)

for GOARCH in "${arch[@]}"; do
    export GOARCH
    for GOOS in "${os[@]}"; do
        export GOOS
        NAME="$BASENAME-$GOOS-$GOARCH"
        echo "Building ${NAME}"
        packr2 build \
            -v \
            -o bin/$NAME \
            -ldflags="-X github.com/bfmiv/klarista/cmd.Version=${KLARISTA_CLI_VERSION}"
    done
done
