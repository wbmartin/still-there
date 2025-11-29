package model

import (
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestUserGrantModel(t *testing.T) {
	suite.Run(t, &UserGrantModelTestSuite{})
}

type UserGrantModelTestSuite struct {
	suite.Suite
}

func (s *UserGrantModelTestSuite) TestGrantConstants() {
	// Test that grant constants are defined and have expected values
	s.Equal(1001, GrantExampleGet)
	s.Equal(1002, GrantExampleAdd)
	s.Equal(1003, GrantExampleChange)
	s.Equal(1004, GrantExampleDelete)
	s.Equal(8001, GrantChangeOwnPassword)
	s.Equal(9001, GrantAddUser)
	s.Equal(9002, GrantBlockUser)
	s.Equal(9003, GrantChangeOtherUserPassword)
}

func (s *UserGrantModelTestSuite) TestAllKnownGrantsMap() {
	// Test that AllKnownGrants map contains all expected grants
	expectedGrants := map[string]int{
		"GET_EXAMPLE":       1001,
		"ADD_EXAMPLE":       1002,
		"CHG_EXAMPLE":       1003,
		"DEL_EXAMPLE":       1004,
		"CHG_OWN_PW":        8001,
		"ADD_USER":          9001,
		"BLOCK_USER":        9002,
		"CHG_OTHER_USER_PW": 9003,
	}

	s.Equal(expectedGrants, AllKnownGrants)

	// Test that all expected keys exist
	s.Contains(AllKnownGrants, "GET_EXAMPLE")
	s.Contains(AllKnownGrants, "ADD_EXAMPLE")
	s.Contains(AllKnownGrants, "CHG_EXAMPLE")
	s.Contains(AllKnownGrants, "DEL_EXAMPLE")
	s.Contains(AllKnownGrants, "CHG_OWN_PW")
	s.Contains(AllKnownGrants, "ADD_USER")
	s.Contains(AllKnownGrants, "BLOCK_USER")
	s.Contains(AllKnownGrants, "CHG_OTHER_USER_PW")
}

func (s *UserGrantModelTestSuite) TestGrantIDToShortCodeMap() {
	// Test that GrantIDToShortCodeMap contains all expected mappings
	expectedMappings := map[int]string{
		1001: "GET_EXAMPLE",
		1002: "ADD_EXAMPLE",
		1003: "CHG_EXAMPLE",
		1004: "DEL_EXAMPLE",
		8001: "CHG_OWN_PW",
		9001: "ADD_USER",
		9002: "BLOCK_USER",
		9003: "CHG_OTHER_USER_PW",
	}

	s.Equal(expectedMappings, GrantIDToShortCodeMap)

	// Test bidirectional mapping consistency
	for id, shortCode := range GrantIDToShortCodeMap {
		s.Equal(id, AllKnownGrants[shortCode])
	}

	for shortCode, id := range AllKnownGrants {
		s.Equal(shortCode, GrantIDToShortCodeMap[id])
	}
}

func (s *UserGrantModelTestSuite) TestUserGrantModelJSON() {
	// Test UserGrantModel JSON marshaling/unmarshaling
	grant := UserGrantModel{
		UserGrantID:      1001,
		GrantShortCode:   "GET_EXAMPLE",
		GrantDescription: "Retrieve examples",
		SecurityContext:  0,
		CreateTS:         1234567890.123,
		CreateUserID:     1,
		UpdateTS:         1234567890.456,
		UpdateUserID:     1,
	}

	// Test marshaling
	data, err := json.Marshal(grant)
	s.NoError(err)
	s.Contains(string(data), `"user_grant_id":1001`)
	s.Contains(string(data), `"grant_short_code":"GET_EXAMPLE"`)
	s.Contains(string(data), `"grant_description":"Retrieve examples"`)

	// Test unmarshaling
	var unmarshaled UserGrantModel
	err = json.Unmarshal(data, &unmarshaled)
	s.NoError(err)
	s.Equal(grant.UserGrantID, unmarshaled.UserGrantID)
	s.Equal(grant.GrantShortCode, unmarshaled.GrantShortCode)
	s.Equal(grant.GrantDescription, unmarshaled.GrantDescription)
}

func (s *UserGrantModelTestSuite) TestInitFunctionExecution() {
	// The init function should run when the package is imported
	// We can verify this by checking that the maps are populated
	// (which they should be since init runs automatically)

	s.NotNil(AllKnownGrants)
	s.NotNil(GrantIDToShortCodeMap)
	s.Greater(len(AllKnownGrants), 0)
	s.Greater(len(GrantIDToShortCodeMap), 0)
}
