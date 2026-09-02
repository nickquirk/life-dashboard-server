package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings_ApplyPatchRejectsUnknownField(t *testing.T) {
	s := DefaultSettings()
	err := s.ApplyPatch([]byte(`{"nope":1}`))
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSettings_ApplyPatchRejectsMalformedJSON(t *testing.T) {
	s := DefaultSettings()
	err := s.ApplyPatch([]byte(`{`))
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestSettings_ApplyPatchIgnoresClientVersion(t *testing.T) {
	s := DefaultSettings()
	require.NoError(t, s.ApplyPatch([]byte(`{"version":99}`)))
	assert.Equal(t, currentSettingsVersion, s.Version)
}

func TestSettings_ValueScanRoundTrip(t *testing.T) {
	in := Settings{Version: currentSettingsVersion}

	v, err := in.Value()
	require.NoError(t, err)

	var out Settings
	require.NoError(t, out.Scan(v))

	assert.Equal(t, in, out)
}

func TestSettings_ScanNilAndEmpty(t *testing.T) {
	var s Settings
	require.NoError(t, s.Scan(nil))
	assert.Equal(t, DefaultSettings(), s)

	s = Settings{}
	require.NoError(t, s.Scan(""))
	assert.Equal(t, DefaultSettings(), s)
}

func TestSettings_ScanRejectsCorruptDocument(t *testing.T) {
	var s Settings
	err := s.Scan([]byte("not json"))
	assert.Error(t, err)
}

// Rows written before the calendar settings were removed still carry the key.
// Scan must ignore it rather than fail the request.
func TestSettings_ScanIgnoresRetiredKeys(t *testing.T) {
	var s Settings
	require.NoError(t, s.Scan([]byte(`{"version":1,"calendar":{"startHour":9}}`)))

	assert.Equal(t, currentSettingsVersion, s.Version)
}
