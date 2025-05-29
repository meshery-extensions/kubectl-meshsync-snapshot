#!/bin/bash

echo "build"
go build -o bin/meshsync-snapshot cmd/meshsync/*.go

echo "tar"
tar -czvf bin/v0.0.1.tar.gz bin/meshsync-snapshot

echo "sha256"
sha256sum bin/v0.0.1.tar.gz