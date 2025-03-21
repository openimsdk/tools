package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSimpleValidator_ValidateSuccess
func TestSimpleValidator_ValidateSuccess(t *testing.T) {
	validator := NewSimpleValidator()
	type Config struct {
		Name string
		Age  int
	}
	config := Config{Name: "Test", Age: 1}

	err := validator.Validate(config)
	assert.Nil(t, err)
}

// TestSimpleValidator_ValidateFailure
func TestSimpleValidator_ValidateFailure(t *testing.T) {
	validator := NewSimpleValidator()
	type Config struct {
		Name string
		Age  int
	}
	config := Config{Name: "", Age: 0}

	err := validator.Validate(config)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestSimpleValidator_ValidateNonStruct
func TestSimpleValidator_ValidateNonStruct(t *testing.T) {
	validator := NewSimpleValidator()
	config := "I am not a struct"

	err := validator.Validate(config)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "validation failed: config must be a struct or a pointer to struct")
}
