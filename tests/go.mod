module github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests

go 1.24

toolchain go1.24.7

require (
	github.com/onsi/ginkgo/v2 v2.25.3
	github.com/onsi/gomega v1.38.2
	github.com/stretchr/testify v1.11.1
	github.com/prometheus/client_golang v1.23.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.4
	k8s.io/api v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	k8s.io/metrics v0.29.3
	sigs.k8s.io/controller-runtime v0.22.1
	sigs.k8s.io/kustomize/api v0.20.1
	sigs.k8s.io/yaml v1.6.0
	gopkg.in/yaml.v2 v2.4.0
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator v0.0.0-00010101000000-000000000000
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1 v0.0.0-00010101000000-000000000000
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client v0.0.0-00010101000000-000000000000
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security v0.0.0-00010101000000-000000000000
	github.com/GoogleContainerTools/kpt/porch/api v0.0.0-20240427025202-5cbd3cbd9237
)

replace (
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator => ../adapters/vnf-operator
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1 => ../adapters/vnf-operator/api/v1alpha1
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/cn-dms => ../cn-dms
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator => ../orchestrator
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security => ../pkg/security
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/ran-dms => ../ran-dms
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn => ../tn
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1 => ../tn/manager/api/v1alpha1
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client => ../o2-client
)