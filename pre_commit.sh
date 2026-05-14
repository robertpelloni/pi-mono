#!/bin/bash
# Pre-commit checks
echo "Running pre-commit checks..."
go fmt ./...
go build ./...
go test $(go list ./... | grep -v 'submodules')
echo "Pre-commit checks completed."
