package domain

import (
	"fmt"
	"strings"
)

const (
	maxNoteTitleLen       = 200
	maxNoteContentLen     = 50_000
	maxNoteColorLen       = 32
	maxNoteItemContentLen = 5_000
)

var validNoteTypes = map[string]struct{}{
	NoteTypeText:      {},
	NoteTypeChecklist: {},
	NoteTypeBullet:    {},
}

func (r *CreateNoteRequest) Validate() error {
	r.Title = strings.TrimSpace(r.Title)
	if len(r.Title) > maxNoteTitleLen {
		return fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxNoteTitleLen)
	}
	if _, ok := validNoteTypes[r.Type]; !ok {
		return fmt.Errorf("%w: type must be one of text, checklist, bullet", ErrInvalidInput)
	}
	if r.Type == NoteTypeText && len(r.Content) > maxNoteContentLen {
		return fmt.Errorf("%w: content must be %d characters or fewer", ErrInvalidInput, maxNoteContentLen)
	}
	if len(r.Color) > maxNoteColorLen {
		return fmt.Errorf("%w: color must be %d characters or fewer", ErrInvalidInput, maxNoteColorLen)
	}
	return nil
}

func (r *UpdateNoteRequest) Validate() error {
	if r.Title != nil {
		*r.Title = strings.TrimSpace(*r.Title)
		if len(*r.Title) > maxNoteTitleLen {
			return fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxNoteTitleLen)
		}
	}
	if r.Type != nil {
		if _, ok := validNoteTypes[*r.Type]; !ok {
			return fmt.Errorf("%w: type must be one of text, checklist, bullet", ErrInvalidInput)
		}
	}
	if r.Content != nil && len(*r.Content) > maxNoteContentLen {
		return fmt.Errorf("%w: content must be %d characters or fewer", ErrInvalidInput, maxNoteContentLen)
	}
	if r.Color != nil && len(*r.Color) > maxNoteColorLen {
		return fmt.Errorf("%w: color must be %d characters or fewer", ErrInvalidInput, maxNoteColorLen)
	}
	return nil
}

func (r *CreateNoteItemRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return fmt.Errorf("%w: content is required", ErrInvalidInput)
	}
	if len(r.Content) > maxNoteItemContentLen {
		return fmt.Errorf("%w: content must be %d characters or fewer", ErrInvalidInput, maxNoteItemContentLen)
	}
	if r.Position < 0 {
		return fmt.Errorf("%w: position must be >= 0", ErrInvalidInput)
	}
	return nil
}

func (r *UpdateNoteItemRequest) Validate() error {
	if r.Content != nil && len(*r.Content) > maxNoteItemContentLen {
		return fmt.Errorf("%w: content must be %d characters or fewer", ErrInvalidInput, maxNoteItemContentLen)
	}
	if r.Position != nil && *r.Position < 0 {
		return fmt.Errorf("%w: position must be >= 0", ErrInvalidInput)
	}
	return nil
}
