package utils

import (
	"encoding/json"
	"fmt"
)

// MapToStruct converts a map to a struct using JSON marshaling/unmarshaling
func MapToStruct(m map[string]interface{}, v interface{}) error {
	if m == nil {
		return fmt.Errorf("input map is nil")
	}
	
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map to JSON: %w", err)
	}
	
	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
	}
	
	return nil
}