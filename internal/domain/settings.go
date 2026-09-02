package domain

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// currentSettingsVersion is bumped whenever the stored shape changes in a way
// that needs translating on read. See migrateSettingsDoc.
const currentSettingsVersion = 1

// Settings is the whole per-user preference document, stored as a single JSON
// column. Adding a field here needs no DB migration: rows written before the
// field existed don't carry the key and pick up its value from
// DefaultSettings() when they're read.
//
// Group new settings under a named sub-struct rather than adding them at the
// top level, so the document stays navigable as the app grows.
type Settings struct {
	Version int `json:"version"`
}

// DefaultSettings is the single source of truth for defaults. Every field must
// be set explicitly, including ones whose default is the zero value, because
// Scan and ApplyPatch both decode on top of the value it returns.
func DefaultSettings() Settings {
	return Settings{
		Version: currentSettingsVersion,
	}
}

// Validate checks the document before it's stored. There is nothing to check
// today - the hook stays so a new setting can validate itself without the
// service having to change.
func (s *Settings) Validate() error {
	return nil
}

// ApplyPatch merges a partial JSON document into s. Keys absent from the patch
// are left alone, which is what lets PATCH /settings work without a pointer
// field per setting.
func (s *Settings) ApplyPatch(patch []byte) error {
	dec := json.NewDecoder(bytes.NewReader(patch))
	dec.DisallowUnknownFields()
	if err := dec.Decode(s); err != nil {
		return fmt.Errorf("%w: malformed settings payload", ErrInvalidInput)
	}
	s.Version = currentSettingsVersion // not client-settable
	return nil
}

// Scan implements sql.Scanner. Decoding over DefaultSettings() is deliberate:
// it's what backfills newly added settings for rows written before they existed.
func (s *Settings) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case nil:
		*s = DefaultSettings()
		return nil
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("settings: cannot scan %T", src)
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		*s = DefaultSettings()
		return nil
	}

	raw, err := migrateSettingsDoc(raw)
	if err != nil {
		return err
	}

	out := DefaultSettings()
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

func (s Settings) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil // string, not []byte — MySQL treats []byte as a blob
}

// GormDBDataType picks a real JSON column on MySQL and TEXT elsewhere, since
// the repository tests run against SQLite.
func (Settings) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	if db.Dialector.Name() == "mysql" {
		return "JSON"
	}
	return "TEXT"
}

// migrateSettingsDoc upgrades a stored document to the current version. It's a
// no-op today; the hook exists so regrouping or renaming a key later doesn't
// strand rows written by an older deploy. A document from a newer version
// (mid-rollback) is passed through — unknown keys are ignored and missing ones
// fall back to defaults, which beats failing the request.
func migrateSettingsDoc(raw []byte) ([]byte, error) {
	var probe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	return raw, nil
}
