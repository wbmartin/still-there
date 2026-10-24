package model

// Grant constants (centralized for all entities and application-wide).
//
// Convention for NEW ids: one block of ten per entity, starting at a
// multiple of ten no shipped grant uses. N+0 Read, N+1 Create, N+2 Update,
// N+3 Delete, N+4..N+9 extras. User and security entities use 9xxx,
// others 1xxx upward. Shipped ids are never renumbered: they are stored in
// group rows and pinned by testdata/grants.json (B054), so the existing
// Example (1001-1004) and User (9001 Create, 9002 Read, ...) layouts stay.
// See doc/810-create-new-entity-type-notes.md.
const (
	// Application-wide grants
	GrantLogon = 1

	// Example entity grants
	GrantExampleRead   = 1001
	GrantExampleCreate = 1002
	GrantExampleUpdate = 1003
	GrantExampleDelete = 1004

	// Device monitoring (migration 0008): the Pi, its bands, the people
	// texted about it, its alarms and the tenant's reminder settings.
	GrantDeviceRead          = 1010
	GrantDeviceCreate        = 1011
	GrantDeviceUpdate        = 1012
	GrantDeviceDelete        = 1013
	GrantDeviceIssueToken    = 1014
	GrantBeaconRead          = 1020
	GrantBeaconCreate        = 1021
	GrantBeaconUpdate        = 1022
	GrantBeaconDelete        = 1023
	GrantContactRead         = 1030
	GrantContactCreate       = 1031
	GrantContactUpdate       = 1032
	GrantContactDelete       = 1033
	GrantAlertRead           = 1040
	GrantAlertUpdate         = 1042 // acknowledge, turn off
	GrantTenantSettingRead   = 1050
	GrantTenantSettingUpdate = 1052

	// User grants
	GrantChangeOwnPassword = 8001

	//Admin User Grants
	GrantUserCreate              = 9001
	GrantUserRead                = 9002
	GrantUserUpdate              = 9003
	GrantUserDelete              = 9004
	GrantUserBlock               = 9008
	GrantChangeOtherUserPassword = 9009

	// User group grants
	GrantUserGroupRead       = 9010
	GrantUserGroupCreate     = 9011
	GrantUserGroupUpdate     = 9012
	GrantUserGroupDelete     = 9013
	GrantAddUserToGroup      = 9014
	GrantRemoveUserFromGroup = 9015

	// Stripping another user's second factor is its own authority, apart
	// from editing the user (9003).
	GrantResetUserMFA = 9020
)

// AllKnownGrants maps grant short codes to their IDs (centralized)
var AllKnownGrants = map[string]int{
	"LOGON":                  GrantLogon,
	"READ_EXAMPLE":           GrantExampleRead,
	"CREATE_EXAMPLE":         GrantExampleCreate,
	"UPDATE_EXAMPLE":         GrantExampleUpdate,
	"DELETE_EXAMPLE":         GrantExampleDelete,
	"READ_DEVICE":            GrantDeviceRead,
	"CREATE_DEVICE":          GrantDeviceCreate,
	"UPDATE_DEVICE":          GrantDeviceUpdate,
	"DELETE_DEVICE":          GrantDeviceDelete,
	"ISSUE_DEVICE_TOKEN":     GrantDeviceIssueToken,
	"READ_BEACON":            GrantBeaconRead,
	"CREATE_BEACON":          GrantBeaconCreate,
	"UPDATE_BEACON":          GrantBeaconUpdate,
	"DELETE_BEACON":          GrantBeaconDelete,
	"READ_CONTACT":           GrantContactRead,
	"CREATE_CONTACT":         GrantContactCreate,
	"UPDATE_CONTACT":         GrantContactUpdate,
	"DELETE_CONTACT":         GrantContactDelete,
	"READ_ALERT":             GrantAlertRead,
	"UPDATE_ALERT":           GrantAlertUpdate,
	"READ_TENANT_SETTING":    GrantTenantSettingRead,
	"UPDATE_TENANT_SETTING":  GrantTenantSettingUpdate,
	"CHG_OWN_PW":             GrantChangeOwnPassword,
	"CREATE_USER":            GrantUserCreate,
	"READ_USER":              GrantUserRead,
	"UPDATE_USER":            GrantUserUpdate,
	"DELETE_USER":            GrantUserDelete,
	"BLOCK_USER":             GrantUserBlock,
	"CHANGE_USER_PW":         GrantChangeOtherUserPassword,
	"READ_USER_GROUP":        GrantUserGroupRead,
	"CREATE_USER_GROUP":      GrantUserGroupCreate,
	"UPDATE_USER_GROUP":      GrantUserGroupUpdate,
	"DELETE_USER_GROUP":      GrantUserGroupDelete,
	"ADD_USER_TO_GROUP":      GrantAddUserToGroup,
	"REMOVE_USER_FROM_GROUP": GrantRemoveUserFromGroup,
	"RESET_MFA_TOKEN":        GrantResetUserMFA,
}

// UserGrants maps grant short codes to their IDs for user-related operations
var UserGrants = map[string]int{
	"LOGON":           GrantLogon,
	"CHG_OWN_PW":      GrantChangeOwnPassword,
	"CREATE_USER":     GrantUserCreate,
	"READ_USER":       GrantUserRead,
	"UPDATE_USER":     GrantUserUpdate,
	"DELETE_USER":     GrantUserDelete,
	"BLOCK_USER":      GrantUserBlock,
	"CHANGE_USER_PW":  GrantChangeOtherUserPassword,
	"RESET_MFA_TOKEN": GrantResetUserMFA,
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
	GrantLogon:                   "LOGON",
	GrantExampleRead:             "READ_EXAMPLE",
	GrantExampleCreate:           "CREATE_EXAMPLE",
	GrantExampleUpdate:           "UPDATE_EXAMPLE",
	GrantExampleDelete:           "DELETE_EXAMPLE",
	GrantDeviceRead:              "READ_DEVICE",
	GrantDeviceCreate:            "CREATE_DEVICE",
	GrantDeviceUpdate:            "UPDATE_DEVICE",
	GrantDeviceDelete:            "DELETE_DEVICE",
	GrantDeviceIssueToken:        "ISSUE_DEVICE_TOKEN",
	GrantBeaconRead:              "READ_BEACON",
	GrantBeaconCreate:            "CREATE_BEACON",
	GrantBeaconUpdate:            "UPDATE_BEACON",
	GrantBeaconDelete:            "DELETE_BEACON",
	GrantContactRead:             "READ_CONTACT",
	GrantContactCreate:           "CREATE_CONTACT",
	GrantContactUpdate:           "UPDATE_CONTACT",
	GrantContactDelete:           "DELETE_CONTACT",
	GrantAlertRead:               "READ_ALERT",
	GrantAlertUpdate:             "UPDATE_ALERT",
	GrantTenantSettingRead:       "READ_TENANT_SETTING",
	GrantTenantSettingUpdate:     "UPDATE_TENANT_SETTING",
	GrantChangeOwnPassword:       "CHG_OWN_PW",
	GrantUserCreate:              "CREATE_USER",
	GrantUserRead:                "READ_USER",
	GrantUserUpdate:              "UPDATE_USER",
	GrantUserDelete:              "DELETE_USER",
	GrantUserBlock:               "BLOCK_USER",
	GrantChangeOtherUserPassword: "CHANGE_USER_PW",
	GrantUserGroupRead:           "READ_USER_GROUP",
	GrantUserGroupCreate:         "CREATE_USER_GROUP",
	GrantUserGroupUpdate:         "UPDATE_USER_GROUP",
	GrantUserGroupDelete:         "DELETE_USER_GROUP",
	GrantAddUserToGroup:          "ADD_USER_TO_GROUP",
	GrantRemoveUserFromGroup:     "REMOVE_USER_FROM_GROUP",
	GrantResetUserMFA:            "RESET_MFA_TOKEN",
}
