package pki

import "testing"

func TestCmdrIdentity(t *testing.T) {
	want := "spiffe://latis/cmdr"
	got := CmdrIdentity()
	if got != want {
		t.Errorf("CmdrIdentity() = %q, want %q", got, want)
	}
}

func TestUnitIdentity(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"abc123", "spiffe://latis/unit/abc123"},
		{"", "spiffe://latis/unit/"},
		{"my-unit-id", "spiffe://latis/unit/my-unit-id"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := UnitIdentity(tt.id)
			if got != tt.want {
				t.Errorf("UnitIdentity(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
