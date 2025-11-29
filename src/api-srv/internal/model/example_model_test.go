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
		SecurityContext:   1,
		CreateTS:          1234567890.123,
		CreateUserID:      1,
		UpdateTS:          1234567890.456,
		UpdateUserID:      1,
	}

	data, err := json.Marshal(example)
	s.NoError(err)
	s.NotEmpty(data)

	// Verify key fields are present
	jsonStr := string(data)
	s.Contains(jsonStr, `"example_id":1`)
	s.Contains(jsonStr, `"example_text":"Test Example"`)
	s.Contains(jsonStr, `"example_int":42`)
	s.Contains(jsonStr, `"example_float":3.14`)
	s.Contains(jsonStr, `"example_int_array":[1,2,3]`)
	s.Contains(jsonStr, `"security_context":1`)
}

func (s *ExampleModelTestSuite) TestExampleModelJSONUnmarshal() {
	jsonData := `{
		"example_id": 1,
		"example_text": "Test Example",
		"example_int": 42,
		"example_float": 3.14,
		"example_int_array": [1, 2, 3],
		"example_float_array": [1.1, 2.2, 3.3],
		"security_context": 1,
		"create_ts": 1234567890.123,
		"create_user_id": 1,
		"update_ts": 1234567890.456,
		"update_user_id": 1
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
		"example_text": "Minimal Example",
		"security_context": 1,
		"create_ts": 1234567890.123,
		"create_user_id": 1,
		"update_ts": 1234567890.456,
		"update_user_id": 1
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
	s.Contains(jsonStr, `"delete_ts":1234567891.789`)
	s.Contains(jsonStr, `"delete_user_id":2`)
}
