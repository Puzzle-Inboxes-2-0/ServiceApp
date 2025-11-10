package config

import (
	"os"
	"regexp"
	"strings"
)

const defaultValueRegex = `\$\{(.*?):(.*?)\}`

// getenv retrieves an environment variable or returns a default value
func getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// evaluateString replaces ${VAR:default} patterns with environment variables
func evaluateString(envStr string) string {
	evaluatedString := envStr
	matches := regexp.MustCompile(defaultValueRegex).FindAllStringSubmatch(evaluatedString, -1)
	
	for _, match := range matches {
		findString := match[0]
		envVar := match[1]
		defaultValue := match[2]
		evaluatedString = strings.ReplaceAll(evaluatedString, findString, getenv(envVar, defaultValue))
	}
	
	// Expand any remaining environment variables
	evaluatedString = os.ExpandEnv(evaluatedString)
	
	return evaluatedString
}

