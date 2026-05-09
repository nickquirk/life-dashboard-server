package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullableDate_JSONNull(t *testing.T) {
	var nd NullableDate
	err := json.Unmarshal([]byte("null"), &nd)
	require.NoError(t, err)
	assert.False(t, nd.Valid)
}

func TestNullableDate_EmptyString(t *testing.T) {
	var nd NullableDate
	err := json.Unmarshal([]byte(`""`), &nd)
	require.NoError(t, err)
	assert.False(t, nd.Valid)
}

func TestNullableDate_ValidYYYYMMDD(t *testing.T) {
	var nd NullableDate
	err := json.Unmarshal([]byte(`"2024-06-15"`), &nd)
	require.NoError(t, err)
	assert.True(t, nd.Valid)
	assert.Equal(t, 2024, nd.Time.Year())
	assert.Equal(t, 6, int(nd.Time.Month()))
	assert.Equal(t, 15, nd.Time.Day())
}

func TestNullableDate_ValidRFC3339(t *testing.T) {
	var nd NullableDate
	err := json.Unmarshal([]byte(`"2024-06-15T10:30:00Z"`), &nd)
	require.NoError(t, err)
	assert.True(t, nd.Valid)
	assert.Equal(t, 2024, nd.Time.Year())
}

func TestNullableDate_InvalidDateString(t *testing.T) {
	var nd NullableDate
	err := json.Unmarshal([]byte(`"not-a-date"`), &nd)
	assert.Error(t, err)
}
