package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// ConnectorConfig ...
type ConnectorConfig struct {
	Name                         string `json:"name,omitempty"`
	ConnectorClass               string `json:"connector.class"`
	DocumentType                 string `json:"type.name"`
	Topics                       string `json:"topics"`
	TopicIndexMap                string `json:"topic.index.map"`
	BatchSize                    string `json:"batch.size"`
	ConnectionURL                string `json:"connection.url"`
	KeyIgnore                    string `json:"key.ignore"`
	SchemaIgnore                 string `json:"schema.ignore"`
	Type                         string `json:"type,omitempty"`
	BehaviorOnMalformedDocuments string `json:"behavior.on.malformed.documents,omitempty"`
}

// ESSinkConnectorSpec defines the desired state of ESSinkConnector
type ESSinkConnectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Config ConnectorConfig `json:"connector.config"`
}

// ESSinkConnectorStatus defines the observed state of ESSinkConnector
type ESSinkConnectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Name   string `json:"name"`
	Topics string `json:"topics"`
	Tasks  int    `json:"tasks"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ESSinkConnector is the Schema for the essinkconnectors API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=essinkconnectors,scope=Namespaced
type ESSinkConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ESSinkConnectorSpec   `json:"spec,omitempty"`
	Status ESSinkConnectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ESSinkConnectorList contains a list of ESSinkConnector
type ESSinkConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ESSinkConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ESSinkConnector{}, &ESSinkConnectorList{})
}
