package model

// Grant constants (centralized for all entities and application-wide)
const (
	// Application-wide grants
	GrantLogon = 1

	// Example entity grants
	GrantExampleRead   = 1001
	GrantExampleCreate = 1002
	GrantExampleUpdate = 1003
	GrantExampleDelete = 1004

	// User grants
	GrantChangeOwnPassword = 8001

	//Admin User Grants
	GrantUserCreate         = 9001
	GrantUserRead           = 9002
	GrantUserUpdate         = 9003
	GrantUserDelete         = 9004
	GrantUserBlock          = 9008
	GrantUserChangePassword = 9009

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
	"LOGON":                  GrantLogon,
	"READ_EXAMPLE":           GrantExampleRead,
	"CREATE_EXAMPLE":         GrantExampleCreate,
	"UPDATE_EXAMPLE":         GrantExampleUpdate,
	"DELETE_EXAMPLE":         GrantExampleDelete,
	"CHG_OWN_PW":             GrantChangeOwnPassword,
	"CREATE_USER":            GrantUserCreate,
	"READ_USER":              GrantUserRead,
	"UPDATE_USER":            GrantUserUpdate,
	"DELETE_USER":            GrantUserDelete,
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
	"LOGON":          GrantLogon,
	"CHG_OWN_PW":     GrantChangeOwnPassword,
	"CREATE_USER":    GrantUserCreate,
	"READ_USER":      GrantUserRead,
	"UPDATE_USER":    GrantUserUpdate,
	"DELETE_USER":    GrantUserDelete,
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
	GrantLogon:               "LOGON",
	GrantExampleRead:         "READ_EXAMPLE",
	GrantExampleCreate:       "CREATE_EXAMPLE",
	GrantExampleUpdate:       "UPDATE_EXAMPLE",
	GrantExampleDelete:       "DELETE_EXAMPLE",
	GrantChangeOwnPassword:   "CHG_OWN_PW",
	GrantUserCreate:          "CREATE_USER",
	GrantUserRead:            "READ_USER",
	GrantUserUpdate:          "UPDATE_USER",
	GrantUserDelete:          "DELETE_USER",
	GrantUserBlock:           "BLOCK_USER",
	GrantUserChangePassword:  "CHANGE_USER_PW",
	GrantUserGroupRead:       "READ_USER_GROUP",
	GrantUserGroupCreate:     "CREATE_USER_GROUP",
	GrantUserGroupUpdate:     "UPDATE_USER_GROUP",
	GrantUserGroupDelete:     "DELETE_USER_GROUP",
	GrantAddUserToGroup:      "ADD_USER_TO_GROUP",
	GrantRemoveUserFromGroup: "REMOVE_USER_FROM_GROUP",
}
