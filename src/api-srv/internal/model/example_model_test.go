package model

import (
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestExampleModel(t *testing.T) {
	suite.Run(t, &ExampleModelTestSuite{})
}

type ExampleModelTestSuite struct {
	suite.Suite
}

func (s *ExampleModelTestSuite) TestExampleModelJSONMarshal() {
	example := ExampleModel{
		ExampleID:         1,
		ExampleText:       "Test Example",
		ExampleInt:        &[]int{42}[0],
		ExampleFloat:      &[]float64{3.14}[0],
		ExampleIntArray:   []int{1, 2, 3},
		ExampleFloatArray: []float64{1.1, 2.2, 3.3},
		ExampleUUID:       "123e4567-e89b-42d3-a456-426614174000",
		ExampleJSONB:      json.RawMessage(`{"k":[1,2]}`),
		SecurityContext:   1,
		CreateTS:          1234567890.123,
		CreateUserID:      1,
		UpdateTS:          1234567890.456,
		UpdateUserID:      1,
	}

	data, err := json.Marshal(example)
	s.NoError(err)
	s.NotEmpty(data)
	// The jsonb value is emitted inline, not as an escaped string.
	s.Contains(string(data), `"exampleUuid":"123e4567-e89b-42d3-a456-426614174000"`)
	s.Contains(string(data), `"exampleJsonb":{"k":[1,2]}`)

	// Verify key fields are present
	jsonStr := string(data)
	s.Contains(jsonStr, `"exampleId":1`)
	s.Contains(jsonStr, `"exampleText":"Test Example"`)
	s.Contains(jsonStr, `"exampleInt":42`)
	s.Contains(jsonStr, `"exampleFloat":3.14`)
	s.Contains(jsonStr, `"exampleIntArray":[1,2,3]`)
	s.Contains(jsonStr, `"securityContext":1`)
}

func (s *ExampleModelTestSuite) TestExampleModelJSONUnmarshal() {
	jsonData := `{
		"exampleId": 1,
		"exampleText": "Test Example",
		"exampleInt": 42,
		"exampleFloat": 3.14,
		"exampleIntArray": [1, 2, 3],
		"exampleFloatArray": [1.1, 2.2, 3.3],
		"securityContext": 1,
		"createTs": 1234567890.123,
		"createUserId": 1,
		"updateTs": 1234567890.456,
		"updateUserId": 1
	}`

	var example ExampleModel
	err := json.Unmarshal([]byte(jsonData), &example)
	s.NoError(err)

	s.Equal(1, example.ExampleID)
	s.Equal("Test Example", example.ExampleText)
	s.NotNil(example.ExampleInt)
	s.Equal(42, *example.ExampleInt)
	s.NotNil(example.ExampleFloat)
	s.Equal(3.14, *example.ExampleFloat)
	s.Equal([]int{1, 2, 3}, example.ExampleIntArray)
	s.Equal([]float64{1.1, 2.2, 3.3}, example.ExampleFloatArray)
	s.Equal(1, example.SecurityContext)
	s.Equal(1234567890.123, example.CreateTS)
	s.Equal(1, example.CreateUserID)
	s.Equal(1234567890.456, example.UpdateTS)
	s.Equal(1, example.UpdateUserID)
}

func (s *ExampleModelTestSuite) TestExampleModelOptionalFields() {
	// Test with null/empty optional fields
	jsonData := `{
		"exampleText": "Minimal Example",
		"securityContext": 1,
		"createTs": 1234567890.123,
		"createUserId": 1,
		"updateTs": 1234567890.456,
		"updateUserId": 1
	}`

	var example ExampleModel
	err := json.Unmarshal([]byte(jsonData), &example)
	s.NoError(err)

	s.Equal("Minimal Example", example.ExampleText)
	s.Nil(example.ExampleInt)
	s.Nil(example.ExampleFloat)
	s.Empty(example.ExampleIntArray)
	s.Empty(example.ExampleFloatArray)
}

func (s *ExampleModelTestSuite) TestDataWrapper() {
	wrapper := DataWrapper{
		Data: "test data",
	}

	data, err := json.Marshal(wrapper)
	s.NoError(err)

	var unmarshaled DataWrapper
	err = json.Unmarshal(data, &unmarshaled)
	s.NoError(err)
	s.Equal("test data", unmarshaled.Data)
}

func (s *ExampleModelTestSuite) TestExampleModelWithDeleteFields() {
	example := ExampleModel{
		ExampleID:       1,
		ExampleText:     "Test Example",
		SecurityContext: 1,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.456,
		UpdateUserID:    1,
		DeleteTS:        &[]float64{1234567891.789}[0],
		DeleteUserID:    &[]int{2}[0],
	}

	data, err := json.Marshal(example)
	s.NoError(err)

	jsonStr := string(data)
	s.Contains(jsonStr, `"deleteTs":1234567891.789`)
	s.Contains(jsonStr, `"deleteUserId":2`)
}

func (s *ExampleModelTestSuite) TestExampleModelJSONBAbsentAndUUIDShape() {
	// No jsonb: the key is omitted, not emitted as null.
	data, err := json.Marshal(ExampleModel{ExampleText: "x"})
	s.NoError(err)
	s.NotContains(string(data), "exampleJsonb")

	// A client body's jsonb arrives verbatim as raw bytes.
	var decoded ExampleModel
	s.NoError(json.Unmarshal([]byte(`{"exampleText":"x","exampleJsonb":[true,null]}`), &decoded))
	s.JSONEq(`[true,null]`, string(decoded.ExampleJSONB))

	// NewUUID is canonical v4: 36 chars, version nibble 4, variant 8-b.
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		u := NewUUID()
		s.Regexp(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, u)
		s.False(seen[u], "duplicate uuid")
		seen[u] = true
	}
}
