package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings_ScanAppliesDefaultsForMissingKeys(t *testing.T) {
	var s Settings
	require.NoError(t, s.Scan([]byte(`{"version":1,"calendar":{"startHour":9}}`)))

	assert.Equal(t, 9, s.Calendar.StartHour)
	assert.Equal(t, DefaultCalendarEndHour, s.Calendar.EndHour)
	assert.True(t, s.Calendar.DynamicRange)
}

func TestSettings_ApplyPatchLeavesUntouchedKeys(t *testing.T) {
	s := Settings{
		Version: currentSettingsVersion,
		Calendar: CalendarSettings{
			StartHour:    9,
			EndHour:      17,
			DynamicRange: true,
		},
	}

	require.NoError(t, s.ApplyPatch([]byte(`{"calendar":{"dynamicRange":false}}`)))

	assert.Equal(t, 9, s.Calendar.StartHour)
	assert.Equal(t, 17, s.Calendar.EndHour)
	assert.False(t, s.Calendar.DynamicRange)
}

func TestSettings_ApplyPatchRejectsUnknownField(t *testing.T) {
	s := DefaultSettings()
	err := s.ApplyPatch([]byte(`{"calendar":{"nope":1}}`))
	assert.ErrorIs(t, err, ErrInvalidInput)

	s = DefaultSettings()
	err = s.ApplyPatch([]byte(`{"nope":1}`))
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
	in := Settings{
		Version: currentSettingsVersion,
		Calendar: CalendarSettings{
			StartHour:    0,
			EndHour:      24,
			DynamicRange: false,
		},
	}

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

func TestSettings_Validate(t *testing.T) {
	cases := map[string]struct {
		settings Settings
		wantErr  bool
	}{
		"valid": {
			settings: DefaultSettings(),
			wantErr:  false,
		},
		"end before start": {
			settings: Settings{Calendar: CalendarSettings{StartHour: 18, EndHour: 9}},
			wantErr:  true,
		},
		"end equals start": {
			settings: Settings{Calendar: CalendarSettings{StartHour: 9, EndHour: 9}},
			wantErr:  true,
		},
		"negative start": {
			settings: Settings{Calendar: CalendarSettings{StartHour: -1, EndHour: 12}},
			wantErr:  true,
		},
		"end past 24": {
			settings: Settings{Calendar: CalendarSettings{StartHour: 9, EndHour: 25}},
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.settings.Validate()
			if tc.wantErr {
				assert.True(t, errors.Is(err, ErrInvalidInput))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
