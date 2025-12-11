package model

// Grant constants (centralized for all entities and application-wide)
const (
	// Example entity grants
	GrantExampleRead   = 1001
	GrantExampleCreate = 1002
	GrantExampleUpdate = 1003
	GrantExampleDelete = 1004

	// Application-wide grants
	GrantChangeOwnPassword  = 8001
	GrantUserCreate         = 9001
	GrantUserBlock          = 9002
	GrantUserChangePassword = 9003

	// User group grants
	GrantUserGroupRead       = 9010
	GrantUserGroupCreate     = 9011
	GrantUserGroupUpdate     = 9012
	GrantUserGroupDelete     = 9013
	GrantAddUserToGroup      = 9014
	GrantRemoveUserFromGroup = 9015
)

// AllKnownGrants maps grant short codes to their IDs (centralized)
var AllKnownGrants = map[string]int{
	"READ_EXAMPLE":           GrantExampleRead,
	"CREATE_EXAMPLE":         GrantExampleCreate,
	"UPDATE_EXAMPLE":         GrantExampleUpdate,
	"DELETE_EXAMPLE":         GrantExampleDelete,
	"CHG_OWN_PW":             GrantChangeOwnPassword,
	"CREATE_USER":            GrantUserCreate,
	"BLOCK_USER":             GrantUserBlock,
	"CHANGE_USER_PW":         GrantUserChangePassword,
	"READ_USER_GROUP":        GrantUserGroupRead,
	"CREATE_USER_GROUP":      GrantUserGroupCreate,
	"UPDATE_USER_GROUP":      GrantUserGroupUpdate,
	"DELETE_USER_GROUP":      GrantUserGroupDelete,
	"ADD_USER_TO_GROUP":      GrantAddUserToGroup,
	"REMOVE_USER_FROM_GROUP": GrantRemoveUserFromGroup,
}

// UserGrants maps grant short codes to their IDs for user-related operations
var UserGrants = map[string]int{
	"CHG_OWN_PW":     GrantChangeOwnPassword,
	"CREATE_USER":    GrantUserCreate,
	"BLOCK_USER":     GrantUserBlock,
	"CHANGE_USER_PW": GrantUserChangePassword,
}

// ExampleGrants maps grant short codes to their IDs for the Example entity
var ExampleGrants = map[string]int{
	"READ_EXAMPLE":   GrantExampleRead,
	"CREATE_EXAMPLE": GrantExampleCreate,
	"UPDATE_EXAMPLE": GrantExampleUpdate,
	"DELETE_EXAMPLE": GrantExampleDelete,
}

// GrantIDToShortCodeMap maps grant IDs to their short codes (centralized)
var GrantIDToShortCodeMap = map[int]string{
	GrantExampleRead:         "READ_EXAMPLE",
	GrantExampleCreate:       "CREATE_EXAMPLE",
	GrantExampleUpdate:       "UPDATE_EXAMPLE",
	GrantExampleDelete:       "DELETE_EXAMPLE",
	GrantChangeOwnPassword:   "CHG_OWN_PW",
	GrantUserCreate:          "CREATE_USER",
	GrantUserBlock:           "BLOCK_USER",
	GrantUserChangePassword:  "CHANGE_USER_PW",
	GrantUserGroupRead:       "READ_USER_GROUP",
	GrantUserGroupCreate:     "CREATE_USER_GROUP",
	GrantUserGroupUpdate:     "UPDATE_USER_GROUP",
	GrantUserGroupDelete:     "DELETE_USER_GROUP",
	GrantAddUserToGroup:      "ADD_USER_TO_GROUP",
	GrantRemoveUserFromGroup: "REMOVE_USER_FROM_GROUP",
}
