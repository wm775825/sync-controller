package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Registry struct {
	URL    string `json:"url,omitempty"`
	Weight int    `json:"weight,omitempty"`
}

// SimageSpec defines the desired state of Simage
type SimageSpec struct {
	ImageId string `json:"imageId,omitempty"`
	// Registries are the desired registries where simage stored.
	Registries []Registry `json:"registries,omitempty"`
	// Important: Run "make" to regenerate code after modifying this file
}

type SimageStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Simage represents an image in the registry.
type Simage struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ObjectMeta	`json:"metadata,omitempty"`

	Spec SimageSpec		`json:"spec,omitempty"`
	Status SimageStatus	`json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SimageList contains a list of Simage.
type SimageList struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ListMeta		`json:"metadata,omitempty"`

	Items []Simage		`json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Sfunction represents a function.
type Sfunction struct {
	metav1.TypeMeta			`json:",inline"`
	metav1.ObjectMeta		`json:"metadata,omitempty"`

	Spec SFunctionSpec		`json:"spec,omitempty"`
	Status SFunctionStatus	`json:"status,omitempty"`
}

// SFunctionSpec defines the desired state of Sfunction
type SFunctionSpec struct {
	Name string			`json:"name,omitempty"`
	Image string		`json:"image,omitempty"`
	PrefetchFunc string	`json:"prefetchFunc,omitempty"`
}

type SFunctionStatus struct {}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SfunctionList contains a list of Sfunction.
type SfunctionList struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ListMeta		`json:"metadata,omitempty"`

	Items []Sfunction	`json:"items"`
}

