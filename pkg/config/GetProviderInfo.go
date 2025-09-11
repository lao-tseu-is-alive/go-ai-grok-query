package config

import (
	"fmt"
	"os"
)

// GetProviderInfoFilePathFromEnv returns the name of the file path to use for loading the providers models if the above env var is defined
// PROVIDER_INFO_FILEPATH : if exist should contain a string with the file path to the json or this function will use the passed default
func GetProviderInfoFilePathFromEnv(defaultName string) string {
	val, exist := os.LookupEnv("PROVIDER_INFO_FILEPATH")
	if !exist {
		return defaultName
	}
	return fmt.Sprintf("%s", val)
}
