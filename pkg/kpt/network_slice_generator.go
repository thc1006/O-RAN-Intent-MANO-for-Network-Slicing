package kpt

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GenerateNetworkSliceFunction generates complete network slice configurations
type GenerateNetworkSliceFunction struct{}

// NewGenerateNetworkSliceFunction creates a new GenerateNetworkSliceFunction
func NewGenerateNetworkSliceFunction() *GenerateNetworkSliceFunction {
	return &GenerateNetworkSliceFunction{}
}

// Run executes the network slice generation function
func (f *GenerateNetworkSliceFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.ResourceList.FunctionConfig == nil {
		return nil, fmt.Errorf("function config is required")
	}

	// Extract slice specification
	sliceSpec, err := f.extractSliceSpec(input.ResourceList.FunctionConfig)
	if err != nil {
		return nil, err
	}

	// Generate resources
	generatedResources := []*yaml.RNode{}

	// 1. Generate NetworkSlice resource
	networkSlice := f.generateNetworkSlice(sliceSpec)
	generatedResources = append(generatedResources, networkSlice)

	// 2. Generate Network Function deployments
	if nfs, exists := sliceSpec["networkFunctions"]; exists {
		if nfList, ok := nfs.([]interface{}); ok {
			for _, nf := range nfList {
				if nfMap, ok := nf.(map[string]interface{}); ok {
					deployment := f.generateNFDeployment(sliceSpec["name"].(string), nfMap)
					if deployment != nil {
						generatedResources = append(generatedResources, deployment)
					}

					service := f.generateNFService(sliceSpec["name"].(string), nfMap)
					if service != nil {
						generatedResources = append(generatedResources, service)
					}
				}
			}
		}
	}

	// 3. Generate MEC deployment if edge computing is enabled
	if edgeComputing, exists := sliceSpec["edgeComputing"]; exists {
		if edgeMap, ok := edgeComputing.(map[string]interface{}); ok {
			if enabled, ok := edgeMap["enabled"].(bool); ok && enabled {
				mecDeployment := f.generateMECDeployment(sliceSpec["name"].(string), edgeMap)
				if mecDeployment != nil {
					generatedResources = append(generatedResources, mecDeployment)
				}
			}
		}
	}

	// 4. Generate ConfigMap for slice configuration
	configMap := f.generateSliceConfigMap(sliceSpec)
	generatedResources = append(generatedResources, configMap)

	// Combine with existing resources
	allResources := append(input.ResourceList.Items, generatedResources...)

	return &FunctionOutput{
		ResourceList: &ResourceList{
			Items:          allResources,
			FunctionConfig: input.ResourceList.FunctionConfig,
		},
	}, nil
}

func (f *GenerateNetworkSliceFunction) extractSliceSpec(config *yaml.RNode) (map[string]interface{}, error) {
	sliceSpecNode, err := config.Pipe(yaml.Lookup("sliceSpec"))
	if err != nil || sliceSpecNode == nil {
		return nil, fmt.Errorf("sliceSpec not found in config")
	}

	parsed := f.parseYAMLNode(sliceSpecNode.YNode())
	if parsedMap, ok := parsed.(map[string]interface{}); ok {
		return parsedMap, nil
	}

	return nil, fmt.Errorf("sliceSpec is not a valid map")
}

func (f *GenerateNetworkSliceFunction) parseYAMLNode(node *yaml.Node) interface{} {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.ScalarNode:
		// Try to parse as number or boolean
		if intVal, err := strconv.Atoi(node.Value); err == nil {
			return intVal
		}
		if floatVal, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return floatVal
		}
		if boolVal, err := strconv.ParseBool(node.Value); err == nil {
			return boolVal
		}
		return node.Value

	case yaml.SequenceNode:
		result := []interface{}{}
		for _, child := range node.Content {
			result = append(result, f.parseYAMLNode(child))
		}
		return result

	case yaml.MappingNode:
		result := make(map[string]interface{})
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i].Value
			value := f.parseYAMLNode(node.Content[i+1])
			result[key] = value
		}
		return result

	default:
		return nil
	}
}

func (f *GenerateNetworkSliceFunction) generateNetworkSlice(spec map[string]interface{}) *yaml.RNode {
	name := spec["name"].(string)
	sliceType := spec["type"].(string)

	// Build sites YAML
	sitesYAML := ""
	if sites, exists := spec["sites"]; exists {
		if siteList, ok := sites.([]interface{}); ok {
			for _, site := range siteList {
				if siteMap, ok := site.(map[string]interface{}); ok {
					sitesYAML += fmt.Sprintf("\n  - siteID: \"%s\"", siteMap["siteID"])
					if cells, ok := siteMap["cells"].([]interface{}); ok {
						sitesYAML += "\n    cells:"
						for _, cell := range cells {
							sitesYAML += fmt.Sprintf("\n    - \"%s\"", cell)
						}
					}
				}
			}
		}
	}

	// Build QoS YAML
	qosYAML := ""
	if qos, exists := spec["qos"]; exists {
		if qosMap, ok := qos.(map[string]interface{}); ok {
			qosYAML = "\n  qos:"
			for key, value := range qosMap {
				qosYAML += fmt.Sprintf("\n    %s: %v", key, value)
			}
		}
	}

	sliceYAML := fmt.Sprintf(`
apiVersion: ran.o-ran.org/v1alpha1
kind: NetworkSlice
metadata:
  name: %s
spec:
  sliceType: %s`, name, sliceType)

	if tenant, exists := spec["tenant"]; exists {
		sliceYAML += fmt.Sprintf("\n  tenant: %s", tenant)
	}

	if sitesYAML != "" {
		sliceYAML += "\n  sites:" + sitesYAML
	}

	sliceYAML += qosYAML

	return yaml.MustParse(sliceYAML)
}

func (f *GenerateNetworkSliceFunction) generateNFDeployment(sliceName string, nf map[string]interface{}) *yaml.RNode {
	nfType := strings.ToLower(nf["type"].(string))
	replicas := 1
	if r, exists := nf["replicas"]; exists {
		if intVal, ok := r.(int); ok {
			replicas = intVal
		}
	}

	deploymentYAML := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s-%s
  labels:
    slice: %s
    nf-type: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s-%s
  template:
    metadata:
      labels:
        app: %s-%s
        slice: %s
        nf-type: %s
    spec:
      containers:
      - name: %s
        image: 5g-core/%s:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 38412
          name: ngap
        resources:
          requests:
            cpu: "100m"
            memory: "256Mi"
          limits:
            cpu: "500m"
            memory: "1Gi"
`, nfType, sliceName, sliceName, nfType, replicas, nfType, sliceName, nfType, sliceName, sliceName, nfType, nfType, nfType)

	return yaml.MustParse(deploymentYAML)
}

func (f *GenerateNetworkSliceFunction) generateNFService(sliceName string, nf map[string]interface{}) *yaml.RNode {
	nfType := strings.ToLower(nf["type"].(string))

	serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s-%s-svc
  labels:
    slice: %s
    nf-type: %s
spec:
  selector:
    app: %s-%s
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: ngap
    port: 38412
    targetPort: 38412
  type: ClusterIP
`, nfType, sliceName, sliceName, nfType, nfType, sliceName)

	return yaml.MustParse(serviceYAML)
}

func (f *GenerateNetworkSliceFunction) generateMECDeployment(sliceName string, edgeConfig map[string]interface{}) *yaml.RNode {
	mecConfig := edgeConfig["mec"].(map[string]interface{})
	location := mecConfig["location"].(string)

	resourcesYAML := ""
	if resources, exists := mecConfig["resources"]; exists {
		if resMap, ok := resources.(map[string]interface{}); ok {
			resourcesYAML = "\n        resources:\n          limits:"
			for key, value := range resMap {
				resourcesYAML += fmt.Sprintf("\n            %s: %v", key, value)
			}
		}
	}

	mecYAML := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mec-%s
  labels:
    slice: %s
    type: mec
    location: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mec-%s
  template:
    metadata:
      labels:
        app: mec-%s
        slice: %s
        type: mec
    spec:
      nodeSelector:
        edge-location: %s
      containers:
      - name: mec-server
        image: edge-computing/mec:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics%s
`, sliceName, sliceName, location, sliceName, sliceName, sliceName, location, resourcesYAML)

	return yaml.MustParse(mecYAML)
}

func (f *GenerateNetworkSliceFunction) generateSliceConfigMap(spec map[string]interface{}) *yaml.RNode {
	name := spec["name"].(string)

	configYAML := fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
data:
  slice-type: "%s"
  tenant: "%s"
`, name, spec["type"], spec["tenant"])

	if qos, exists := spec["qos"]; exists {
		if qosMap, ok := qos.(map[string]interface{}); ok {
			for key, value := range qosMap {
				configYAML += fmt.Sprintf("  qos-%s: \"%v\"\n", key, value)
			}
		}
	}

	return yaml.MustParse(configYAML)
}
