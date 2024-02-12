#!/bin/sh

go build -ldflags="-X main.Version=$(git rev-parse --short HEAD)" .

echo "If compilation succeeded you may want to run 'cp protoc-gen-go-psm ~/go/bin/' for local use."
