package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestUserGrantModel(t *testing.T) {
	suite.Run(t, &UserGrantModelTestSuite{})
}

type UserGrantModelTestSuite struct {
	suite.Suite
}

func (s *UserGrantModelTestSuite) TestGrantConstants() {
	// Test that grant constants are defined and have expected values
	s.Equal(1, GrantLogon)
	s.Equal(1001, GrantExampleRead)
	s.Equal(1002, GrantExampleCreate)
	s.Equal(1003, GrantExampleUpdate)
	s.Equal(1004, GrantExampleDelete)
	s.Equal(8001, GrantChangeOwnPassword)
	s.Equal(9001, GrantUserCreate)
	s.Equal(9002, GrantUserRead)
	s.Equal(9003, GrantUserUpdate)
	s.Equal(9004, GrantUserDelete)
	s.Equal(9008, GrantUserBlock)
	s.Equal(9009, GrantChangeOtherUserPassword)
	s.Equal(9020, GrantResetUserMFA)
}

func (s *UserGrantModelTestSuite) TestAllKnownGrantsMap() {
	// Test that AllKnownGrants map contains all expected grants
	expectedGrants := map[string]int{
		"LOGON":                  1,
		"READ_EXAMPLE":           1001,
		"CREATE_EXAMPLE":         1002,
		"UPDATE_EXAMPLE":         1003,
		"DELETE_EXAMPLE":         1004,
		"READ_DEVICE":            1010,
		"CREATE_DEVICE":          1011,
		"UPDATE_DEVICE":          1012,
		"DELETE_DEVICE":          1013,
		"ISSUE_DEVICE_TOKEN":     1014,
		"READ_BEACON":            1020,
		"CREATE_BEACON":          1021,
		"UPDATE_BEACON":          1022,
		"DELETE_BEACON":          1023,
		"READ_CONTACT":           1030,
		"CREATE_CONTACT":         1031,
		"UPDATE_CONTACT":         1032,
		"DELETE_CONTACT":         1033,
		"READ_ALERT":             1040,
		"UPDATE_ALERT":           1042,
		"READ_TENANT_SETTING":    1050,
		"UPDATE_TENANT_SETTING":  1052,
		"CHG_OWN_PW":             8001,
		"CREATE_USER":            9001,
		"READ_USER":              9002,
		"UPDATE_USER":            9003,
		"DELETE_USER":            9004,
		"BLOCK_USER":             9008,
		"CHANGE_USER_PW":         9009,
		"READ_USER_GROUP":        9010,
		"CREATE_USER_GROUP":      9011,
		"UPDATE_USER_GROUP":      9012,
		"DELETE_USER_GROUP":      9013,
		"ADD_USER_TO_GROUP":      9014,
		"REMOVE_USER_FROM_GROUP": 9015,
		"RESET_MFA_TOKEN":        9020,
	}

	s.Equal(expectedGrants, AllKnownGrants)

	// Test that all expected keys exist
	s.Contains(AllKnownGrants, "READ_EXAMPLE")
	s.Contains(AllKnownGrants, "CREATE_EXAMPLE")
	s.Contains(AllKnownGrants, "UPDATE_EXAMPLE")
	s.Contains(AllKnownGrants, "DELETE_EXAMPLE")
	s.Contains(AllKnownGrants, "CHG_OWN_PW")
	s.Contains(AllKnownGrants, "CREATE_USER")
	s.Contains(AllKnownGrants, "BLOCK_USER")
	s.Contains(AllKnownGrants, "CHANGE_USER_PW")
}

func (s *UserGrantModelTestSuite) TestGrantIDToShortCodeMap() {
	// Test that GrantIDToShortCodeMap contains all expected mappings
	expectedMappings := map[int]string{
		1:    "LOGON",
		1001: "READ_EXAMPLE",
		1002: "CREATE_EXAMPLE",
		1003: "UPDATE_EXAMPLE",
		1004: "DELETE_EXAMPLE",
		1010: "READ_DEVICE",
		1011: "CREATE_DEVICE",
		1012: "UPDATE_DEVICE",
		1013: "DELETE_DEVICE",
		1014: "ISSUE_DEVICE_TOKEN",
		1020: "READ_BEACON",
		1021: "CREATE_BEACON",
		1022: "UPDATE_BEACON",
		1023: "DELETE_BEACON",
		1030: "READ_CONTACT",
		1031: "CREATE_CONTACT",
		1032: "UPDATE_CONTACT",
		1033: "DELETE_CONTACT",
		1040: "READ_ALERT",
		1042: "UPDATE_ALERT",
		1050: "READ_TENANT_SETTING",
		1052: "UPDATE_TENANT_SETTING",
		8001: "CHG_OWN_PW",
		9001: "CREATE_USER",
		9002: "READ_USER",
		9003: "UPDATE_USER",
		9004: "DELETE_USER",
		9008: "BLOCK_USER",
		9009: "CHANGE_USER_PW",
		9010: "READ_USER_GROUP",
		9011: "CREATE_USER_GROUP",
		9012: "UPDATE_USER_GROUP",
		9013: "DELETE_USER_GROUP",
		9014: "ADD_USER_TO_GROUP",
		9015: "REMOVE_USER_FROM_GROUP",
		9020: "RESET_MFA_TOKEN",
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
		GrantShortCode:   "READ_EXAMPLE",
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
	s.Contains(string(data), `"grant_short_code":"READ_EXAMPLE"`)
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
