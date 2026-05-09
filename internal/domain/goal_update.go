package domain

import (
	"bytes"
	"encoding/json"
)

// GoalUpdate distinguishes "field absent in JSON" from "field present with null"
// from "field present with a value", which standard *Goal unmarshalling collapses.
//
// MUST be used as a value (not pointer) on request structs — see UnmarshalJSON.
type GoalUpdate struct {
	Set   bool  // true iff the JSON contained the "goal" key
	Value *Goal // nil + Set==true means "clear"; non-nil means "set"
}

func (g *GoalUpdate) UnmarshalJSON(data []byte) error {
	// This method only runs when the JSON contained "goal" — that's what makes Set meaningful.
	g.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		g.Value = nil
		return nil
	}

	var v Goal
	if err := json.Unmarshal(data, &v); err != nil {
		return err // malformed object — propagate as a 400
	}
	g.Value = &v
	return nil
}
