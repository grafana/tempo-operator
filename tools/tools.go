//go:build tools

// Package tools pins build-time code generation tools so their versions are
// tracked in go.sum rather than fetched via a throwaway module at build time.
package tools

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
