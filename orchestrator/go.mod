module github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator

go 1.24.0

require (
	github.com/stretchr/testify v1.11.1
	github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security => ../pkg/security
