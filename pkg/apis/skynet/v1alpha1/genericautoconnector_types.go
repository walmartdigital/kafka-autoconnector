package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GenericAutoConnectorSpec defines the desired state of GenericAutoConnector
type GenericAutoConnectorSpec struct {
	Config map[string]string `json:"connector.config"`
}

// GenericAutoConnectorStatus defines the observed state of GenericAutoConnector
type GenericAutoConnectorStatus struct {
	ConnectorName string `json:"connector.name"`
	Topics        string `json:"topics"`
	Tasks         string `json:"tasks"`
	// +nullable
	LastUpdate metav1.Time `json:"last.update,omitempty"`
	Status     string      `json:"status,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GenericAutoConnector is the Schema for the genericautoconnectors API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=genericautoconnectors,scope=Namespaced
type GenericAutoConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GenericAutoConnectorSpec   `json:"spec,omitempty"`
	Status GenericAutoConnectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GenericAutoConnectorList contains a list of GenericAutoConnector
type GenericAutoConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GenericAutoConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GenericAutoConnector{}, &GenericAutoConnectorList{})
}
