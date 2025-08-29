package config

import (
	"fmt"
	"os"
	"unicode/utf8"
)

const (
	minApiKeyLength = 80
)

// GetXaiApiKeyFromEnv returns string with XAI API key from an env variable XAI_API_KEY
func GetXaiApiKeyFromEnv() (string, error) {
	apiKey := ""
	val, exist := os.LookupEnv("XAI_API_KEY")
	if exist {
		apiKey = val
		if utf8.RuneCountInString(apiKey) < minApiKeyLength {
			fmt.Printf("💥💥 ERROR: XAI_API_KEY should contain at least %d characters (got %d).",
				minApiKeyLength, utf8.RuneCountInString(val))
		}
		return fmt.Sprintf("%s", apiKey), nil
	} else {
		fmt.Println("💥💥 ERROR: XAI_API_KEY environment variable not set.")
		fmt.Println("If you don't have one go to : https://console.x.ai/team/default/api-keys")
		fmt.Println("Please set it before running the program:")
		fmt.Println("export XAI_API_KEY='your_api_key_here'")
		return "", fmt.Errorf("error geeting api key from : %s", "XAI_API_KEY")
	}
}

// GetGeminiApiKeyFromEnv returns string with Gemini API key from an env variable GEMINI_API_KEY
func GetGeminiApiKeyFromEnv() (string, error) {
	apiKey := ""
	val, exist := os.LookupEnv("GEMINI_API_KEY")
	if exist {
		apiKey = val
		return fmt.Sprintf("%s", apiKey), nil
	} else {
		fmt.Println("💥💥 ERROR: GEMINI_API_KEY environment variable not set.")
		fmt.Println("Please set it before running the program:")
		fmt.Println("export GEMINI_API_KEY='your_api_key_here'")
		return "", fmt.Errorf("error getting api key from: %s", "GEMINI_API_KEY")
	}
}
