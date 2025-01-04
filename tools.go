//go:build tools
// +build tools

package main

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
