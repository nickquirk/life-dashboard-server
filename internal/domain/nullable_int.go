package domain

import (
	"bytes"
	"encoding/json"
)

// NullableInt distinguishes "field absent in JSON" from "field present with
// null" from "field present with a value", for PATCH requests that must be
// able to clear a value. Same approach as GoalUpdate.
//
// MUST be used as a value (not pointer) on request structs.
type NullableInt struct {
	Set   bool // true iff the JSON contained the key
	Value *int // nil + Set==true means "clear"
}

func (n *NullableInt) UnmarshalJSON(data []byte) error {
	// Only runs when the key is present — that's what makes Set meaningful.
	n.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.Value = nil
		return nil
	}

	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}
