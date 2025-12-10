package model

// ExampleModel represents an example entity
type ExampleModel struct {
	ExampleID         int       `json:"exampleId" db:"example_id"`
	ExampleText       string    `json:"exampleText" db:"example_text"`
	ExampleInt        *int      `json:"exampleInt,omitempty" db:"example_int"`
	ExampleFloat      *float64  `json:"exampleFloat,omitempty" db:"example_float"`
	ExampleIntArray   []int     `json:"exampleIntArray,omitempty" db:"example_int_array"`
	ExampleFloatArray []float64 `json:"exampleFloatArray,omitempty" db:"example_float_array"`
	SecurityContext   int       `json:"securityContext" db:"security_context"`
	CreateTS          float64   `json:"createTs" db:"create_ts"`
	CreateUserID      int       `json:"createUserId" db:"create_user_id"`
	UpdateTS          float64   `json:"updateTs" db:"update_ts"`
	UpdateUserID      int       `json:"updateUserId" db:"update_user_id"`
	DeleteTS          *float64  `json:"deleteTs,omitempty" db:"delete_ts"`
	DeleteUserID      *int      `json:"deleteUserId,omitempty" db:"delete_user_id"`
}
