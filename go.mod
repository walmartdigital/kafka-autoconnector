module github.com/walmartdigital/kafka-autoconnector

go 1.13

require (
	github.com/chinniehendrix/go-kaya v0.0.0-20200508230955-217336014d1e
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.4.0
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/redhat-cop/operator-utils v0.2.4
	github.com/spf13/pflag v1.0.5
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
