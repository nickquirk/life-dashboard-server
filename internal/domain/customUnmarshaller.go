package domain

import (
	"encoding/json"
	"time"
)

// NullableDate acts as a "smart" wrapper for dates in PATCH requests
type NullableDate struct {
	Time  time.Time // The parsed time
	Valid bool      // true = valid time; false = explicit null/clear
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (n *NullableDate) UnmarshalJSON(data []byte) error {
	// 1. Check for explicit JSON null
	if string(data) == "null" {
		n.Valid = false
		return nil
	}

	// Unmarshal into a string to handle quotes/escaping safely
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Handle "Clear" signals: empty string or explicit null
	if str == "" {
		n.Valid = false
		return nil
	}

	// Try parsing.
	// Can support multiple formats here,
	// but sticking to standard YYYY-MM-DD or RFC3339 is safer.
	parsed, err := time.Parse("2006-01-02", str)
	if err != nil {
		// Fallback: Try RFC3339 if frontend sends full ISO
		parsed, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
	}

	n.Time = parsed
	n.Valid = true
	return nil
}
