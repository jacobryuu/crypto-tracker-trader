#!/bin/bash

# Find all Go packages in the current module and run go vet on each
go list -f '{{.Dir}}' ./... | while read -r dir; do
  go vet "$dir" || exit 1
done
