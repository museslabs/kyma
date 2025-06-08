#!/bin/bash
# Build script that injects version from git tag
VERSION=$(git describe --tags --always)
go build -ldflags "-s -w -X github.com/museslabs/kyma/cmd.version=$VERSION" -o kyma main.go

echo "Binary: ./kyma" 