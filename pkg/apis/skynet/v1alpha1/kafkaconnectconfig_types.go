package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaConnectConfigSpec defines the desired state of KafkaConnectConfig
type KafkaConnectConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	KafkaConnectAddress    string `json:"kafka.connect.addr"`
	MaxTaskRestarts        int    `json:"max.task.restarts"`
	MaxConnectorRestarts   int    `json:"max.connector.restarts"`
	MaxConnectorHardResets int    `json:"max.connector.hard.resets"`
}

// KafkaConnectConfigStatus defines the observed state of KafkaConnectConfig
type KafkaConnectConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaConnectConfig is the Schema for the kafkaconnectconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kafkaconnectconfigs,scope=Namespaced
type KafkaConnectConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectConfigSpec   `json:"spec,omitempty"`
	Status KafkaConnectConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaConnectConfigList contains a list of KafkaConnectConfig
type KafkaConnectConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaConnectConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaConnectConfig{}, &KafkaConnectConfigList{})
}
