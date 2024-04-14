#!/bin/sh

# I don't even ..
go env -w GOFLAGS="-buildvcs=false"

# Run golang tests
go test ./...
