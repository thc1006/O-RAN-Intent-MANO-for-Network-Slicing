// Package nephio provides kustomization types
package nephio

// Kustomization represents a kustomization file
type Kustomization struct {
	APIVersion   string            `json:"apiVersion,omitempty"`
	Kind         string            `json:"kind,omitempty"`
	Resources    []string          `json:"resources,omitempty"`
	Images       []Image           `json:"images,omitempty"`
	NamePrefix   string            `json:"namePrefix,omitempty"`
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
}

// Image represents an image transformation in kustomization
type Image struct {
	Name    string `json:"name,omitempty"`
	NewName string `json:"newName,omitempty"`
	NewTag  string `json:"newTag,omitempty"`
}
