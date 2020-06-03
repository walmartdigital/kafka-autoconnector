module github.com/walmartdigital/kafka-autoconnector

go 1.13

require (
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-toolsmith/pkgload v1.0.0 // indirect
	github.com/golang/mock v1.4.3
	github.com/google/go-cmp v0.4.0
	github.com/google/ko v0.4.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mibk/dupl v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/qiniu/checkstyle v0.0.0-20181122073030-e47d31cae315 // indirect
	github.com/quasilyte/go-namecheck v0.0.0-20190530083159-81b081ff1afc // indirect
	github.com/redhat-cop/operator-utils v0.2.4
	github.com/securego/gosec v0.0.0-20200401082031-e946c8c39989 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/walmartdigital/go-kaya v0.0.0-20200602185225-b6b835a560c9
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/tools v0.0.0-20200601175630-2caf76543d99 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.4
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
