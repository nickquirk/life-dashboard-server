package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoalUpdate_Absent(t *testing.T) {
	var req UpdateRoutineRequest
	err := json.Unmarshal([]byte(`{"title":"x"}`), &req)
	assert.NoError(t, err)
	assert.False(t, req.Goal.Set)
	assert.Nil(t, req.Goal.Value)
}

func TestGoalUpdate_Null(t *testing.T) {
	var req UpdateRoutineRequest
	err := json.Unmarshal([]byte(`{"goal":null}`), &req)
	assert.NoError(t, err)
	assert.True(t, req.Goal.Set)
	assert.Nil(t, req.Goal.Value)
}

func TestGoalUpdate_Object(t *testing.T) {
	var req UpdateRoutineRequest
	err := json.Unmarshal([]byte(`{"goal":{"type":"count","target":3}}`), &req)
	assert.NoError(t, err)
	assert.True(t, req.Goal.Set)
	assert.NotNil(t, req.Goal.Value)
	assert.Equal(t, GoalTypeCount, req.Goal.Value.Type)
	assert.Equal(t, 3, req.Goal.Value.Target)
}

func TestGoalUpdate_Malformed(t *testing.T) {
	var req UpdateRoutineRequest
	err := json.Unmarshal([]byte(`{"goal":"oops"}`), &req)
	assert.Error(t, err)
}
