// Package v1alpha1 contains API Schema definitions for the aot v1alpha1 API group.
// +kubebuilder:object:generate=true
// +groupName=aot.uncworks.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "aot.uncworks.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionResource scheme.
	SchemeBuilder = &builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&AgentRun{}, &AgentRunList{})
}

// builder registers this group's types with a scheme.
//
// controller-runtime's scheme.Builder does the same thing and is deprecated for
// an api package, because importing it pulls controller-runtime into anything
// that only wants the types. This keeps the same Register signature, so no call
// site changes, and depends only on apimachinery.
type builder struct {
	GroupVersion schema.GroupVersion
	objects      []runtime.Object
}

// Register adds types to the builder. Every types file calls this from its init.
func (b *builder) Register(objects ...runtime.Object) *builder {
	b.objects = append(b.objects, objects...)
	return b
}

// AddToScheme adds every registered type to the scheme.
func (b *builder) AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(b.GroupVersion, b.objects...)
	metav1.AddToGroupVersion(s, b.GroupVersion)
	return nil
}
