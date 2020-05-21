package v1alpha1

import (
	"github.com/chinniehendrix/go-kaya/pkg/kafkaconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ESSinkConnectorSpec defines the desired state of ESSinkConnector
type ESSinkConnectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Config kafkaconnect.ConnectorConfig `json:"connector.config"`
}

// ESSinkConnectorStatus defines the observed state of ESSinkConnector
type ESSinkConnectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ConnectorName string      `json:"connector.name"`
	Topics        string      `json:"topics"`
	Tasks         int         `json:"tasks"`
	LastUpdate    metav1.Time `json:"last.update"`
	Status        string      `json:"status"`
	Reason        string      `json:"reason"`
}

func (e ESSinkConnector) SetStatus(status ESSinkConnectorStatus) {
	e.Status = status
}

func (e ESSinkConnector) GetStatus() ESSinkConnectorStatus {
	return e.Status
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
