package repository

import (
	"fmt"
	"strings"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type NoteRepository interface {
	CreateNote(domain.Note) (domain.Note, error)
	GetNotesByUserID(userID uint, archived bool) ([]domain.Note, error)
	GetNoteByID(userID, noteID uint) (domain.Note, error)
	UpdateNote(userID, noteID uint, updates map[string]interface{}) error
	DeleteNote(userID, noteID uint) error

	CreateNoteItem(userID uint, item domain.NoteItem) (domain.NoteItem, error)
	UpdateNoteItem(userID, noteID, itemID uint, updates map[string]interface{}) error
	DeleteNoteItem(userID, noteID, itemID uint) error

	ConvertNoteToList(userID, noteID uint, newType string) error
	ConvertNoteToText(userID, noteID uint) error
	ReorderNoteItems(userID, noteID uint, orderedItemIDs []uint) error
}

type GormNoteRepository struct {
	Db *gorm.DB
}

func (r *GormNoteRepository) CreateNote(note domain.Note) (domain.Note, error) {
	err := r.Db.Create(&note).Error
	return note, err
}

func (r *GormNoteRepository) GetNotesByUserID(userID uint, archived bool) ([]domain.Note, error) {
	notes := make([]domain.Note, 0)
	err := r.Db.
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Where("user_id = ? AND is_archived = ?", userID, archived).
		Order("is_pinned DESC, created_at DESC").
		Find(&notes).Error
	return notes, err
}

func (r *GormNoteRepository) GetNoteByID(userID, noteID uint) (domain.Note, error) {
	var note domain.Note
	if err := r.Db.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		return domain.Note{}, gorm.ErrRecordNotFound
	}
	return note, nil
}

func (r *GormNoteRepository) UpdateNote(userID, noteID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.Note{}).Where("id = ? AND user_id = ?", noteID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormNoteRepository) DeleteNote(userID, noteID uint) error {
	result := r.Db.Unscoped().Where("id = ? AND user_id = ?", noteID, userID).Delete(&domain.Note{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormNoteRepository) CreateNoteItem(userID uint, item domain.NoteItem) (domain.NoteItem, error) {
	if _, err := r.GetNoteByID(userID, item.NoteID); err != nil {
		return domain.NoteItem{}, gorm.ErrRecordNotFound
	}
	err := r.Db.Create(&item).Error
	return item, err
}

func (r *GormNoteRepository) UpdateNoteItem(userID, noteID, itemID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.NoteItem{}).
		Where("id = ? AND note_id = ? AND note_id IN (SELECT id FROM notes WHERE user_id = ?)",
			itemID, noteID, userID).
		Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormNoteRepository) DeleteNoteItem(userID, noteID, itemID uint) error {
	result := r.Db.Unscoped().
		Where("id = ? AND note_id = ? AND note_id IN (SELECT id FROM notes WHERE user_id = ?)",
			itemID, noteID, userID).
		Delete(&domain.NoteItem{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormNoteRepository) ConvertNoteToList(userID, noteID uint, newType string) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		var note domain.Note
		if err := tx.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		if note.Type != domain.NoteTypeText {
			return fmt.Errorf("%w: can only convert text notes to %s", domain.ErrInvalidInput, newType)
		}
		var items []domain.NoteItem
		for i, line := range strings.Split(note.Content, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				items = append(items, domain.NoteItem{NoteID: noteID, Content: line, Position: i})
			}
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return tx.Model(&domain.Note{}).Where("id = ?", noteID).Updates(map[string]interface{}{
			"type": newType, "content": "",
		}).Error
	})
}

func (r *GormNoteRepository) ConvertNoteToText(userID, noteID uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		var note domain.Note
		if err := tx.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		if note.Type == domain.NoteTypeText {
			return fmt.Errorf("%w: note is already a text note", domain.ErrInvalidInput)
		}
		var noteItems []domain.NoteItem
		if err := tx.Where("note_id = ?", noteID).Order("position ASC").Find(&noteItems).Error; err != nil {
			return err
		}
		lines := make([]string, len(noteItems))
		for i, item := range noteItems {
			lines[i] = item.Content
		}
		joined := strings.Join(lines, "\n")
		if err := tx.Model(&domain.Note{}).Where("id = ?", noteID).Updates(map[string]interface{}{
			"type": domain.NoteTypeText, "content": joined,
		}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("note_id = ?", noteID).Delete(&domain.NoteItem{}).Error
	})
}

func (r *GormNoteRepository) ReorderNoteItems(userID, noteID uint, orderedItemIDs []uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		var note domain.Note
		if err := tx.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		if len(orderedItemIDs) > 0 {
			var count int64
			if err := tx.Model(&domain.NoteItem{}).
				Where("id IN ? AND note_id = ?", orderedItemIDs, noteID).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(orderedItemIDs)) {
				return fmt.Errorf("%w: some item IDs do not belong to this note", domain.ErrInvalidInput)
			}
		}
		for i, id := range orderedItemIDs {
			if err := tx.Model(&domain.NoteItem{}).Where("id = ?", id).Update("position", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
