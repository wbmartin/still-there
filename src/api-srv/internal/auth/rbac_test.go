package auth

import (
	"bytes"
	"log/slog"
	"testing"

	"stillthere-api/internal/logging"
	"stillthere-api/internal/model"
)

func newTestRBACManager() *RBACManager {
	logger := logging.NewLogger("test", &bytes.Buffer{}, slog.LevelDebug)
	return NewRBACManager(logger, "testuser")
}

func TestRBACManager_HasGrant(t *testing.T) {
	rbac := newTestRBACManager()

	cases := []struct {
		name            string
		claims          *Claims
		grantID         int
		securityContext int
		want            bool
	}{
		{
			name:            "nil claims",
			claims:          nil,
			grantID:         model.GrantExampleRead,
			securityContext: 1,
			want:            false,
		},
		{
			name:            "nil grants",
			claims:          &Claims{SecurityContext: 1, Grants: nil},
			grantID:         model.GrantExampleRead,
			securityContext: 1,
			want:            false,
		},
		{
			name: "empty grants",
			claims: &Claims{
				SecurityContext: 1,
				Grants:          []int{},
			},
			grantID:         model.GrantExampleRead,
			securityContext: 1,
			want:            false,
		},
		{
			name: "grant present in matching context",
			claims: &Claims{
				SecurityContext: 1,
				Grants:          []int{model.GrantExampleRead, model.GrantExampleCreate},
			},
			grantID:         model.GrantExampleRead,
			securityContext: 1,
			want:            true,
		},
		{
			name: "grant present but security context mismatch",
			claims: &Claims{
				SecurityContext: 1,
				Grants:          []int{model.GrantExampleRead},
			},
			grantID:         model.GrantExampleRead,
			securityContext: 2,
			want:            false,
		},
		{
			name: "grant absent in matching context",
			claims: &Claims{
				SecurityContext: 1,
				Grants:          []int{model.GrantExampleCreate},
			},
			grantID:         model.GrantExampleRead,
			securityContext: 1,
			want:            false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rbac.HasGrant(c.claims, c.grantID, c.securityContext)
			if got != c.want {
				t.Errorf("HasGrant() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRBACManager_HasAnyGrantInContext(t *testing.T) {
	rbac := newTestRBACManager()

	cases := []struct {
		name            string
		claims          *Claims
		securityContext int
		want            bool
	}{
		{"nil claims", nil, 1, false},
		{"nil grants", &Claims{SecurityContext: 1, Grants: nil}, 1, false},
		{"empty grants", &Claims{SecurityContext: 1, Grants: []int{}}, 1, false},
		{
			"grants in matching context",
			&Claims{SecurityContext: 1, Grants: []int{model.GrantExampleRead}},
			1,
			true,
		},
		{
			"grants but context mismatch",
			&Claims{SecurityContext: 1, Grants: []int{model.GrantExampleRead}},
			2,
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rbac.HasAnyGrantInContext(c.claims, c.securityContext)
			if got != c.want {
				t.Errorf("HasAnyGrantInContext() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRBACManager_ValidateGrantID(t *testing.T) {
	rbac := newTestRBACManager()

	if !rbac.ValidateGrantID(model.GrantExampleRead) {
		t.Errorf("ValidateGrantID(%d) = false, want true", model.GrantExampleRead)
	}
	if rbac.ValidateGrantID(999999) {
		t.Error("ValidateGrantID(999999) = true, want false for unknown grant id")
	}
}
