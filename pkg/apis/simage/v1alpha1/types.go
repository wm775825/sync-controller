package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SimageSpec defines the desired state of Simage
type SimageSpec struct {
	// Registries are the desired registries where simage stored.
	Registries []string `json:"registries,omitempty"`
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
	metav1.ObjectMeta	`json:"metadata,omitempty"`

	Items []Simage		`json:"items"`
}
