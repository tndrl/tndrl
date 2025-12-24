package pki

import (
	"fmt"
)

const (
	// TrustDomain is the SPIFFE trust domain for Latis.
	TrustDomain = "latis"
)

// NodeIdentity returns the SPIFFE identity URI for a node with the given ID.
func NodeIdentity(id string) string {
	return fmt.Sprintf("spiffe://%s/node/%s", TrustDomain, id)
}

// UnitIdentity is deprecated, use NodeIdentity instead.
// Kept for backwards compatibility during transition.
var UnitIdentity = NodeIdentity
