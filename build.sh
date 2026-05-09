#!/bin/bash
set -e

APP_NAME=mcsim
DIST=dist

trap 'echo ""; echo "=================================="; echo "Build failed!"; echo "=================================="' ERR

mkdir -p "$DIST"

build() {
    GOOS=$1
    GOARCH=$2
    EXT=$3
    OUTDIR="$DIST/$APP_NAME-$GOOS-$GOARCH"
    ARCHIVENAME="$DIST/$APP_NAME-$GOOS-$GOARCH"

    echo ""
    echo "========================="
    echo "Building $GOOS $GOARCH..."
    echo "========================="

    mkdir -p "$OUTDIR"

    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$OUTDIR/$APP_NAME$EXT" || {
        echo "Build failed for $GOOS $GOARCH"
        exit 1
    }

    if [ "$GOOS" = "windows" ]; then
        if command -v zip &>/dev/null; then
            echo "Packing $ARCHIVENAME.zip ..."
            zip -j "$ARCHIVENAME.zip" "$OUTDIR/$APP_NAME$EXT" || {
                echo "Archive failed for $GOOS $GOARCH"
                exit 1
            }
            echo "Created: $ARCHIVENAME.zip"
        else
            echo "[!] zip not found, skipping $ARCHIVENAME.zip (binary saved to $OUTDIR/)"
        fi
    else
        echo "Packing $ARCHIVENAME.tar.gz ..."
        tar -czf "$ARCHIVENAME.tar.gz" -C "$DIST" "$APP_NAME-$GOOS-$GOARCH" || {
            echo "Archive failed for $GOOS $GOARCH"
            exit 1
        }
        echo "Created: $ARCHIVENAME.tar.gz"
    fi
}

build windows amd64 .exe
build linux   amd64
build darwin  amd64
build darwin  arm64

echo ""
echo "=================================="
echo "All builds completed successfully!"
echo "=================================="