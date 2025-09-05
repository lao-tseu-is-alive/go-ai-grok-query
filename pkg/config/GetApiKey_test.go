package config

import (
	"os"
	"strings"
	"testing"
)

func TestGetApiKey(t *testing.T) {
	// A valid dummy key for testing purposes
	validKey := "a_very_long_and_valid_api_key_for_testing_purposes"
	shortKey := "short"
	envVar := "TEST_API_KEY"
	providerName := "TestProvider"

	t.Run("Success", func(t *testing.T) {
		// Set the environment variable for this test case
		t.Setenv(envVar, validKey)

		key, err := getApiKey(envVar, providerName)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if key != validKey {
			t.Errorf("Expected key '%s', but got '%s'", validKey, key)
		}
	})

	t.Run("FailureNotSet", func(t *testing.T) {
		// Ensure the variable is unset
		os.Unsetenv(envVar)

		_, err := getApiKey(envVar, providerName)
		if err == nil {
			t.Error("Expected an error because the environment variable is not set, but got nil")
		}
		expectedErrorMsg := "API key not set"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message to contain '%s', but got '%s'", expectedErrorMsg, err.Error())
		}
	})

	t.Run("FailureTooShort", func(t *testing.T) {
		t.Setenv(envVar, shortKey)

		_, err := getApiKey(envVar, providerName)
		if err == nil {
			t.Error("Expected an error because the API key is too short, but got nil")
		}
		expectedErrorMsg := "must be at least"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message to contain '%s', but got '%s'", expectedErrorMsg, err.Error())
		}
	})
}
