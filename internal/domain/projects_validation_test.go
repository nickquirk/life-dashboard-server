package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validator interface{ Validate() error }

func TestProjectsValidation(t *testing.T) {
	now := time.Now()
	longTitle := strings.Repeat("a", maxProjectTitleLen+1)

	tests := []struct {
		name    string
		req     validator
		wantErr string // "" = valid
	}{
		{"create project ok", &CreateProjectRequest{Title: "Garden", Color: "green"}, ""},
		{"create project blank title", &CreateProjectRequest{Title: "   ", Color: "green"}, "title is required"},
		{"create project long title", &CreateProjectRequest{Title: longTitle, Color: "green"}, "title must be"},
		{"create project missing color", &CreateProjectRequest{Title: "Garden"}, "color"},
		{"create project non-swatch color", &CreateProjectRequest{Title: "Garden", Color: "slate"}, "color"},

		{"update project empty ok", &UpdateProjectRequest{}, ""},
		{"update project blank title", &UpdateProjectRequest{Title: ptrStr(" ")}, "title is required"},
		{"update project bad color", &UpdateProjectRequest{Color: ptrStr("red")}, "color"},
		{"update project good color", &UpdateProjectRequest{Color: ptrStr("indigo")}, ""},

		{"create action ok", &CreateActionRequest{Title: "buy compost"}, ""},
		{"create action with duration", &CreateActionRequest{Title: "x", DurationMins: ptrInt(30)}, ""},
		{"create action blank title", &CreateActionRequest{Title: ""}, "title is required"},
		{"create action long notes", &CreateActionRequest{Title: "x", Notes: strings.Repeat("a", maxActionNotesLen+1)}, "notes"},
		{"create action zero duration", &CreateActionRequest{Title: "x", DurationMins: ptrInt(0)}, "durationMins"},
		{"create action huge duration", &CreateActionRequest{Title: "x", DurationMins: ptrInt(1441)}, "durationMins"},
		{"create action max duration", &CreateActionRequest{Title: "x", DurationMins: ptrInt(1440)}, ""},

		{"update action empty ok", &UpdateActionRequest{}, ""},
		{"update action clear duration ok", &UpdateActionRequest{DurationMins: NullableInt{Set: true}}, ""},
		{"update action bad duration", &UpdateActionRequest{DurationMins: NullableInt{Set: true, Value: ptrInt(-5)}}, "durationMins"},
		{"update action blank title", &UpdateActionRequest{Title: ptrStr("")}, "title is required"},

		{"change stage refine", &ChangeStageRequest{Stage: StageRefine}, ""},
		{"change stage scheduled rejected", &ChangeStageRequest{Stage: StageScheduled}, "stage"},
		{"change stage done rejected", &ChangeStageRequest{Stage: StageDone}, "stage"},
		{"change stage unknown rejected", &ChangeStageRequest{Stage: "nope"}, "stage"},

		{"reorder ready", &ReorderActionsRequest{Stage: StageReady}, ""},
		{"reorder scheduled rejected", &ReorderActionsRequest{Stage: StageScheduled}, "reordered"},

		{"schedule ok", &ScheduleActionRequest{Date: &now}, ""},
		{"schedule no date", &ScheduleActionRequest{}, "date is required"},
		{"move no date", &MoveActionRequest{}, "date is required"},

		{"break down ok", &BreakDownActionRequest{Titles: []string{"a", "b"}}, ""},
		{"break down one title", &BreakDownActionRequest{Titles: []string{"a"}}, "at least"},
		{"break down blank title", &BreakDownActionRequest{Titles: []string{"a", " "}}, "title is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidInput))
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestProjectsValidation_TrimsTitles(t *testing.T) {
	p := CreateProjectRequest{Title: "  Garden  ", Color: "blue"}
	require.NoError(t, p.Validate())
	assert.Equal(t, "Garden", p.Title)

	b := BreakDownActionRequest{Titles: []string{" a ", "b "}}
	require.NoError(t, b.Validate())
	assert.Equal(t, []string{"a", "b"}, b.Titles)
}

func TestStage(t *testing.T) {
	for _, s := range []Stage{StageBraindump, StageRefine, StageReady} {
		assert.True(t, s.Valid())
		assert.True(t, s.IsBoardStage())
	}
	for _, s := range []Stage{StageScheduled, StageDone} {
		assert.True(t, s.Valid())
		assert.False(t, s.IsBoardStage())
	}
	assert.False(t, Stage("nope").Valid())
}

func TestNullableInt_Unmarshal(t *testing.T) {
	var absent, null, set UpdateActionRequest
	require.NoError(t, json.Unmarshal([]byte(`{}`), &absent))
	require.NoError(t, json.Unmarshal([]byte(`{"durationMins":null}`), &null))
	require.NoError(t, json.Unmarshal([]byte(`{"durationMins":45}`), &set))

	assert.False(t, absent.DurationMins.Set)
	assert.True(t, null.DurationMins.Set)
	assert.Nil(t, null.DurationMins.Value)
	assert.True(t, set.DurationMins.Set)
	require.NotNil(t, set.DurationMins.Value)
	assert.Equal(t, 45, *set.DurationMins.Value)

	var bad UpdateActionRequest
	assert.Error(t, json.Unmarshal([]byte(`{"durationMins":"soon"}`), &bad))
}
