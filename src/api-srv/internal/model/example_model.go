package model

// ExampleModel represents an example entity
type ExampleModel struct {
	ExampleID         int       `json:"example_id" db:"example_id"`
	ExampleText       string    `json:"example_text" db:"example_text"`
	ExampleInt        *int      `json:"example_int,omitempty" db:"example_int"`
	ExampleFloat      *float64  `json:"example_float,omitempty" db:"example_float"`
	ExampleIntArray   []int     `json:"example_int_array,omitempty" db:"example_int_array"`
	ExampleFloatArray []float64 `json:"example_float_array,omitempty" db:"example_float_array"`
	SecurityContext   int       `json:"security_context" db:"security_context"`
	CreateTS          float64   `json:"create_ts" db:"create_ts"`
	CreateUserID      int       `json:"create_user_id" db:"create_user_id"`
	UpdateTS          float64   `json:"update_ts" db:"update_ts"`
	UpdateUserID      int       `json:"update_user_id" db:"update_user_id"`
	DeleteTS          *float64  `json:"delete_ts,omitempty" db:"delete_ts"`
	DeleteUserID      *int      `json:"delete_user_id,omitempty" db:"delete_user_id"`
}

// ExampleGrants maps grant short codes to their IDs for the Example entity
// example: The actual constants are defined in user_grant.go for centralization
var ExampleGrants = map[string]int{
	"GET_EXAMPLE": 1001,
	"ADD_EXAMPLE": 1002,
	"CHG_EXAMPLE": 1003,
	"DEL_EXAMPLE": 1004,
}
