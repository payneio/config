package config

import (
	"encoding/json"
	"os"
	"strings"
)

func loadEnvironmentVariables() {

	// walk env variables
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		key := parts[0]
		val := parts[1]

		// if starts with CONFIG
		if strippedKey, ok := stripConfigPrefix(key); ok {

			// if the variable is json, set as JSON
			if isJSON(val) {
				SetJSON(strippedKey, val)
				continue
			}

			// if the variable is a simple string, just use it
			Set(strippedKey, val)
		}
	}
}

func isJSON(s string) bool {

	// It might be json if it is bracketed
	mightBeJSON := false
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "{") {
		mightBeJSON = true
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		mightBeJSON = true
	}

	// To make sure, let's try unmarshalling it
	if mightBeJSON {
		var test interface{}
		err := json.Unmarshal([]byte(s), &test)
		if err == nil {
			return true
		}
	}
	return false
}
