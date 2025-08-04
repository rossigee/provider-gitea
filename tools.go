//go:build tools

package tools

import (
	_ "github.com/crossplane/crossplane-tools/cmd/angryjet"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
