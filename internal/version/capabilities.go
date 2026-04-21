package version

import (
	"github.com/Masterminds/semver/v3"
)

// Feature acts as an enum for togglable Astarte capabilities
type Feature string

const (
	// FDOVault represents the HashiCorp Vault support for FDO
	FDOVault Feature = "FDOVault"
)

// matrix stores the pre-compiled semver constraints for each feature
var matrix map[Feature]*semver.Constraints

// Checker evaluates if a specific Astarte version supports certain features.
type Checker struct {
	version *semver.Version
}

// NewChecker safely parses the Astarte version and returns a Checker.
// It returns an error if the user provided an invalid semantic version.
func NewChecker(versionStr string) (*Checker, error) {
	v, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, err
	}
	return &Checker{version: v}, nil
}

func init() {
	matrix = map[Feature]*semver.Constraints{
		// Supports FDO Vault strictly in 1.4.0 and above
		FDOVault: mustParseConstraint(">= 1.4.0"),
	}
}

// mustParseConstraint is a local helper that panics if a hardcoded constraint is invalid.
func mustParseConstraint(c string) *semver.Constraints {
	constraint, err := semver.NewConstraint(c)
	if err != nil {
		panic("invalid semver constraint in capabilities matrix: " + err.Error())
	}
	return constraint
}

// Supports evaluates if the configured version satisfies the feature's constraint.
func (c *Checker) Supports(f Feature) bool {
	constraint, exists := matrix[f]
	if !exists {
		// Default to false if a feature isn't in the matrix
		return false
	}
	return constraint.Check(c.version)
}
