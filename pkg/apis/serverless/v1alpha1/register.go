package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/wm775825/sync-controller/pkg/apis/serverless"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{
	Group:   serverless.GroupName,
	Version: "v1alpha1",
}

func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownType(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Simage{}, &SimageList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownType)
	AddToScheme = SchemeBuilder.AddToScheme
)


