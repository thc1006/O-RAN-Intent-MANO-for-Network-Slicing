package porch

import (
	"time"
)

// PackageStatus represents the lifecycle status of a package
type PackageStatus string

const (
	PackageStatusDraft     PackageStatus = "draft"
	PackageStatusProposed  PackageStatus = "proposed"
	PackageStatusPublished PackageStatus = "published"
	PackageStatusDeprecated PackageStatus = "deprecated"
)

// PackageSpec defines the specification for creating a new package
type PackageSpec struct {
	Name        string
	Namespace   string
	Repository  string
	Description string
	Version     string
	Resources   []Resource
	Dependencies []Dependency
}

// Package represents a Porch package with its metadata and resources
type Package struct {
	UID         string
	Name        string
	Namespace   string
	Repository  string
	Description string
	Version     string
	Status      PackageStatus
	Resources   []Resource
	Dependencies []Dependency
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

// Resource represents a file/resource within a package
type Resource struct {
	Name    string
	Content string
	Type    string
}

// Dependency represents a package dependency
type Dependency struct {
	Name    string
	Version string
	Optional bool
}

// ResolvedDependency represents a resolved package dependency
type ResolvedDependency struct {
	Name      string
	Version   string
	Package   *Package
	Resolved  bool
	ResolutionPath []string
}

// ValidationResult contains the results of package validation
type ValidationResult struct {
	IsValid            bool
	Errors             []string
	Warnings           []string
	ValidatedResources []string
	ValidationTime     time.Time
}

// RollbackResult contains the results of a package rollback operation
type RollbackResult struct {
	CurrentVersion    string
	PreviousVersion   string
	CurrentPackageUID string
	RollbackTime      time.Time
	Success           bool
}

// PromotionPolicy defines rules for package promotion
type PromotionPolicy struct {
	RequireValidation bool
	RequireApproval   bool
	AllowedTransitions map[PackageStatus][]PackageStatus
}

// DefaultPromotionPolicy returns the default promotion policy
func DefaultPromotionPolicy() *PromotionPolicy {
	return &PromotionPolicy{
		RequireValidation: true,
		RequireApproval:   false,
		AllowedTransitions: map[PackageStatus][]PackageStatus{
			PackageStatusDraft:     {PackageStatusProposed},
			PackageStatusProposed:  {PackageStatusPublished, PackageStatusDraft},
			PackageStatusPublished: {PackageStatusDeprecated},
			PackageStatusDeprecated: {},
		},
	}
}