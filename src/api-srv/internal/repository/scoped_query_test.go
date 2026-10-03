package repository

import (
	"reflect"
	"testing"
)

func TestBuildScopedWhere(t *testing.T) {
	cases := []struct {
		name            string
		securityContext int
		afterUpdateTs   float64
		idColumn        string
		idList          []int
		wantWhere       string
		wantArgs        []any
	}{
		{
			name:            "no ids, no afterUpdateTs",
			securityContext: 3,
			afterUpdateTs:   0,
			idColumn:        "example_id",
			idList:          nil,
			wantWhere:       ` WHERE security_context_id = $1 AND delete_user_id IS NULL`,
			wantArgs:        []any{3},
		},
		{
			name:            "no ids, with afterUpdateTs",
			securityContext: 3,
			afterUpdateTs:   1700000000,
			idColumn:        "example_id",
			idList:          nil,
			wantWhere: ` WHERE security_context_id = $1 AND delete_user_id IS NULL` +
				` AND update_ts > $2`,
			wantArgs: []any{3, 1700000000.0},
		},
		{
			name:            "id list ignores afterUpdateTs",
			securityContext: 3,
			afterUpdateTs:   1700000000,
			idColumn:        "example_id",
			idList:          []int{10, 20},
			wantWhere: ` WHERE security_context_id = $1 AND delete_user_id IS NULL` +
				` AND example_id IN ($2, $3)`,
			wantArgs: []any{3, 10, 20},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotWhere, gotArgs := buildScopedWhere(
				c.securityContext, c.afterUpdateTs, c.idColumn, c.idList,
			)
			if gotWhere != c.wantWhere {
				t.Errorf("where = %q, want %q", gotWhere, c.wantWhere)
			}
			if !reflect.DeepEqual(gotArgs, c.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, c.wantArgs)
			}
			if len(gotArgs) == 0 {
				t.Fatalf("expected at least the security context arg")
			}
		})
	}
}

func TestWithLiveParent(t *testing.T) {
	base := ` WHERE security_context_id = $1 AND delete_user_id IS NULL`
	got := withLiveParent(base, "child_tbl", "parent_fk", "parent_tbl", "parent_id")

	want := base + ` AND EXISTS (SELECT 1 FROM parent_tbl live_parent` +
		` WHERE live_parent.parent_id = child_tbl.parent_fk AND live_parent.delete_user_id IS NULL)`
	if got != want {
		t.Errorf("withLiveParent() = %q, want %q", got, want)
	}
}

func TestBuildParentScopedWhere(t *testing.T) {
	cases := []struct {
		name            string
		securityContext int
		afterUpdateTs   float64
		idColumn        string
		idList          []int
		parentColumn    string
		parentID        int
		wantWhere       string
		wantArgs        []any
	}{
		{
			name:            "no parent filter",
			securityContext: 3,
			idColumn:        "example_id",
			parentColumn:    "parent_id",
			parentID:        0,
			wantWhere:       ` WHERE security_context_id = $1 AND delete_user_id IS NULL`,
			wantArgs:        []any{3},
		},
		{
			name:            "parent filter with no ids",
			securityContext: 3,
			idColumn:        "example_id",
			parentColumn:    "parent_id",
			parentID:        5,
			wantWhere: ` WHERE security_context_id = $1 AND delete_user_id IS NULL` +
				` AND parent_id = $2`,
			wantArgs: []any{3, 5},
		},
		{
			name:            "parent filter placeholder numbered after id list",
			securityContext: 3,
			idColumn:        "example_id",
			idList:          []int{10, 20},
			parentColumn:    "parent_id",
			parentID:        5,
			wantWhere: ` WHERE security_context_id = $1 AND delete_user_id IS NULL` +
				` AND example_id IN ($2, $3)` +
				` AND parent_id = $4`,
			wantArgs: []any{3, 10, 20, 5},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotWhere, gotArgs := buildParentScopedWhere(
				c.securityContext, c.afterUpdateTs, c.idColumn, c.idList,
				c.parentColumn, c.parentID,
			)
			if gotWhere != c.wantWhere {
				t.Errorf("where = %q, want %q", gotWhere, c.wantWhere)
			}
			if !reflect.DeepEqual(gotArgs, c.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, c.wantArgs)
			}
		})
	}
}
