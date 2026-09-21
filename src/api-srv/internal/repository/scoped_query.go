package repository

import (
	"fmt"
	"strings"
)

// buildScopedWhere assembles the WHERE clause every tenant-scoped list read
// needs, and returns it with its argument list. Every entity filters by
// security context, excludes soft-deleted rows, takes an optional id list and
// an optional incremental-fetch timestamp; writing that once per entity is one
// chance per entity to omit the security context.
//
// The rules it encodes, unchanged from the per-repository code it replaces:
//
//   - security_context_id is ALWAYS $1 and is never optional. An id list
//     narrows the result further; it never replaces the tenant filter, so
//     passing another tenant's id returns nothing rather than that tenant's
//     row.
//   - Soft-deleted rows (delete_user_id set) are always excluded.
//   - With no id list, afterUpdateTs > 0 adds `update_ts > $2` for an
//     incremental refresh.
//   - With an id list, the ids bind as $2..$N and afterUpdateTs is ignored:
//     the caller asked for specific rows, so it gets them.
func buildScopedWhere(
	securityContext int,
	afterUpdateTs float64,
	idColumn string,
	idList []int,
) (string, []interface{}) {
	whereClause := ` WHERE security_context_id = $1 AND delete_user_id IS NULL`
	argList := []interface{}{securityContext}

	if len(idList) > 0 {
		placeholderList := make([]string, len(idList))
		for i, id := range idList {
			placeholderList[i] = fmt.Sprintf("$%d", len(argList)+1)
			argList = append(argList, id)
		}
		whereClause += fmt.Sprintf(
			` AND %s IN (%s)`, idColumn, strings.Join(placeholderList, ", "),
		)
		return whereClause, argList
	}

	if afterUpdateTs > 0 {
		whereClause += fmt.Sprintf(` AND update_ts > $%d`, len(argList)+1)
		argList = append(argList, afterUpdateTs)
	}

	return whereClause, argList
}

// withLiveParent restricts a child list to rows whose PARENT is not soft
// deleted: a child is visible only if its parent is visible. Filtering on read
// is reversible, where stamping delete_user_id onto every child (cascade in
// application code) is not.
//
// The child table has to be named explicitly because the FK and the parent's
// primary key usually share a column name: inside the subquery an unqualified
// column binds to the PARENT, so the predicate would compare the parent to
// itself and every row would pass.
func withLiveParent(
	whereClause string,
	childTable string,
	childForeignKeyColumn string,
	parentTable string,
	parentKeyColumn string,
) string {
	return whereClause + fmt.Sprintf(
		` AND EXISTS (SELECT 1 FROM %s live_parent`+
			` WHERE live_parent.%s = %s.%s AND live_parent.delete_user_id IS NULL)`,
		parentTable, parentKeyColumn, childTable, childForeignKeyColumn,
	)
}

// buildParentScopedWhere is buildScopedWhere plus an optional parent filter,
// for entities that hang off another. The parent id is applied on top of the
// tenant filter, never instead of it.
func buildParentScopedWhere(
	securityContext int,
	afterUpdateTs float64,
	idColumn string,
	idList []int,
	parentColumn string,
	parentID int,
) (string, []interface{}) {
	whereClause, argList := buildScopedWhere(
		securityContext, afterUpdateTs, idColumn, idList)
	if parentID > 0 {
		whereClause += fmt.Sprintf(
			` AND %s = $%d`, parentColumn, len(argList)+1)
		argList = append(argList, parentID)
	}
	return whereClause, argList
}
