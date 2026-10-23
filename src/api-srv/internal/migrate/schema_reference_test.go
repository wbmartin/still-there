package migrate

import (
	"context"
	"fmt"
	"strings"
	"testing"

	sqlfiles "stillthere-api/sql"
)

// The reference is what bin/510 writes to doc/003 from an empty database
// taken through every migration. app_user is 0001 plus later ALTERs, so
// a column added by 0003 proves the output is the migrated schema, not
// the first file.
func TestSchemaReferencePrintsTheMigratedSchema(t *testing.T) {
	conn := scratchDatabase(t)
	ctx := context.Background()
	migrationList, err := Load(sqlfiles.Migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(conn, migrationList).Up(ctx); err != nil {
		t.Fatal(err)
	}

	var first strings.Builder
	if err := WriteSchemaReference(ctx, conn, &first); err != nil {
		t.Fatal(err)
	}
	reference := first.String()
	last := migrationList[len(migrationList)-1]
	for _, want := range []string{
		"CREATE TABLE app_user (\n    app_user_id SERIAL NOT NULL,\n",
		"    preference_jsonb jsonb,\n",
		"    user_grant_list integer[] NOT NULL,\n",
		"    create_ts numeric(17,7) NOT NULL,\n",
		"CREATE UNIQUE INDEX idx_example_uuid ON public.example " +
			"USING btree (example_uuid);\n",
		fmt.Sprintf("-- Through migration %04d_%s.\n", last.Version,
			last.Name),
	} {
		if !strings.Contains(reference, want) {
			t.Errorf("reference lacks %q", want)
		}
	}
	if strings.Contains(reference, "CREATE TABLE schema_migration") {
		t.Error("reference includes the migrator's own table")
	}
	// Tables in name order, so the file diffs cleanly.
	if strings.Index(reference, "CREATE TABLE app_user (") >
		strings.Index(reference, "CREATE TABLE example (") {
		t.Error("tables are not sorted by name")
	}

	var second strings.Builder
	if err := WriteSchemaReference(ctx, conn, &second); err != nil {
		t.Fatal(err)
	}
	if second.String() != reference {
		t.Error("two runs on one database printed different text")
	}
}

func TestSchemaReferenceRefusesAnUnmigratedDatabase(t *testing.T) {
	conn := scratchDatabase(t)
	var out strings.Builder
	err := WriteSchemaReference(context.Background(), conn, &out)
	if err == nil || !strings.Contains(err.Error(), "run up first") {
		t.Errorf("expected a run-up-first error, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out.String())
	}
}

// A DEFAULT other than a sequence breaks doc/900, so the reference shows
// it instead of dropping it the way it drops nextval().
func TestSchemaColumnDefinition(t *testing.T) {
	text := func(s string) *string { return &s }
	length := func(n int32) *int32 { return &n }
	for _, tc := range []struct {
		column schemaColumn
		want   string
	}{
		{schemaColumn{Name: "a_id", DataType: "integer", IsNullable: "NO",
			Default: text("nextval('a_a_id_seq'::regclass)")},
			"a_id SERIAL NOT NULL"},
		{schemaColumn{Name: "n", DataType: "character varying",
			IsNullable: "YES", CharLength: length(50)},
			"n varchar(50)"},
		{schemaColumn{Name: "l", DataType: "ARRAY", UdtName: "_float8",
			IsNullable: "YES"}, "l double precision[]"},
		{schemaColumn{Name: "s", DataType: "text", IsNullable: "NO",
			Default: text("'x'::text")}, "s text DEFAULT 'x'::text NOT NULL"},
		{schemaColumn{Name: "t", DataType: "timestamp with time zone",
			IsNullable: "YES"}, "t timestamptz"},
	} {
		if got := tc.column.definition(); got != tc.want {
			t.Errorf("definition() = %q, want %q", got, tc.want)
		}
	}
}
