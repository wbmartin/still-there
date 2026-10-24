// Package sqlfiles carries the schema migrations into the binary.
//
// Embedding, rather than reading a directory at run time, is what lets a
// deploy migrate with exactly the SQL that was built and tested: the migrator
// ships as the same artefact as the API, and there need be no checkout on the
// target host to read from.
package sqlfiles

import "embed"

// Migrations holds migrations/NNNN_name.sql. Version is the leading integer;
// files apply in ascending order, forward only. roles.sql and fixtures/
// are deliberately NOT embedded -- local role setup and test data, not
// schema.
//
//go:embed migrations/*.sql
var Migrations embed.FS
