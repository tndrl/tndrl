package pki

import (
	"fmt"
)

const (
	// TrustDomain is the SPIFFE trust domain for Latis.
	TrustDomain = "latis"
)

// CmdrIdentity returns the SPIFFE identity URI for the cmdr.
func CmdrIdentity() string {
	return fmt.Sprintf("spiffe://%s/cmdr", TrustDomain)
}

// UnitIdentity returns the SPIFFE identity URI for a unit with the given ID.
func UnitIdentity(id string) string {
	return fmt.Sprintf("spiffe://%s/unit/%s", TrustDomain, id)
}
