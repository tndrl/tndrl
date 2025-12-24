package pki

import "testing"

func TestNodeIdentity(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"abc123", "spiffe://latis/node/abc123"},
		{"", "spiffe://latis/node/"},
		{"my-node-id", "spiffe://latis/node/my-node-id"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := NodeIdentity(tt.id)
			if got != tt.want {
				t.Errorf("NodeIdentity(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestUnitIdentityAlias(t *testing.T) {
	// UnitIdentity should be an alias for NodeIdentity
	got := UnitIdentity("test")
	want := NodeIdentity("test")
	if got != want {
		t.Errorf("UnitIdentity should equal NodeIdentity, got %q vs %q", got, want)
	}
}
