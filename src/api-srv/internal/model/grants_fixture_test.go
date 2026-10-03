package model

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// The web app mirrors AllKnownGrants by hand in
// src/web-app/constants/grants.ts. This test keeps testdata/grants.json equal
// to AllKnownGrants; src/web-app/tests/contract.test.ts replays the same file
// against grants.ts, so a grant added on one side only fails a test.
//
// Only bin/305-update-grants-fixture passes -update-grants; a normal run
// never writes the tree.
var updateGrantsFixture = flag.Bool(
	"update-grants", false, "rewrite testdata/grants.json",
)

const grantsFixturePath = "testdata/grants.json"

// A floor so an emptied map cannot pass on both sides.
const grantsFixtureFloor = 18

func TestGrantsFixtureIsCurrent(t *testing.T) {
	if len(AllKnownGrants) < grantsFixtureFloor {
		t.Fatalf("AllKnownGrants has %d entries, floor is %d",
			len(AllKnownGrants), grantsFixtureFloor)
	}

	// encoding/json writes map keys sorted, so the file is stable.
	want, err := json.MarshalIndent(AllKnownGrants, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')

	if *updateGrantsFixture {
		err := os.MkdirAll(filepath.Dir(grantsFixturePath), 0o755)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(grantsFixturePath, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	got, err := os.ReadFile(grantsFixturePath)
	if err != nil {
		t.Fatalf("read %s: %v (run bin/305-update-grants-fixture)",
			grantsFixturePath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale against AllKnownGrants; run "+
			"bin/305-update-grants-fixture, then mirror the change in "+
			"src/web-app/constants/grants.ts", grantsFixturePath)
	}
}
