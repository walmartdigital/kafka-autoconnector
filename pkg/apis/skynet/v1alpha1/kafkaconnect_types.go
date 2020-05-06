package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaConnectSpec defines the desired state of KafkaConnect
type KafkaConnectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	KafkaConnectURL string `json:"kafkaconnect.url"`
}

// KafkaConnectStatus defines the observed state of KafkaConnect
type KafkaConnectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Version          string   `json:"version"`
	Commit           string   `json:"commit"`
	KafkaClusterID   string   `json:"kafka.cluster.id"`
	ConnectorCount   int      `json:"connector.count"`
	ActiveConnectors []string `json:"active.connectors"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaConnect is the Schema for the kafkaconnects API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kafkaconnects,scope=Namespaced
type KafkaConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectSpec   `json:"spec,omitempty"`
	Status KafkaConnectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaConnectList contains a list of KafkaConnect
type KafkaConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaConnect `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaConnect{}, &KafkaConnectList{})
}
