// test/testutil/scheme.go — Shared k8s runtime.Scheme for test packages.
// Every test package that uses a fake k8s client needs the same two schemes
// registered: the standard client-go types and the aot v1alpha1 CRD types.
// This file is the single registration point to avoid repeating the init()
// boilerplate in each package.
package testutil

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
)

// NewScheme returns a new runtime.Scheme with both the standard k8s client-go
// types and the aot v1alpha1 CRD types registered.  Call this once per test
// binary; do not mutate the returned scheme.
func NewScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(aotv1alpha1.AddToScheme(s))
	return s
}
