module github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager

go 1.24.5

replace (
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security => ../../pkg/security
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1 => ./api/v1alpha1
)

require (
	github.com/gorilla/mux v1.8.1
	github.com/prometheus/client_golang v1.23.2
	github.com/stretchr/testify v1.11.1
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security v0.0.0-00010101000000-000000000000
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1 v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	k8s.io/klog/v2 v2.130.1
	sigs.k8s.io/controller-runtime v0.22.1
)