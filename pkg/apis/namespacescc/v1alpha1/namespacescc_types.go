package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NamespaceSCCSpec defines the desired state of NamespaceSCC
// +k8s:openapi-gen=true
type NamespaceSCCSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	UUID        int32 `json:"uuid"`
	SccPriority int32 `json:"sccPriority"`
	// +listType=set
	WhiteList      []string `json:"whiteList"`
	ServiceAccount string   `json:"serviceAccount"`
}

// NamespaceSCCStatus defines the observed state of NamespaceSCC
// +k8s:openapi-gen=true
type NamespaceSCCStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceSCC is the Schema for the namespacesccs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type NamespaceSCC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceSCCSpec   `json:"spec,omitempty"`
	Status NamespaceSCCStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceSCCList contains a list of NamespaceSCC
type NamespaceSCCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceSCC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceSCC{}, &NamespaceSCCList{})
}
