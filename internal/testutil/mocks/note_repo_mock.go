package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockNoteRepository implements repository.NoteRepository with function fields.
// Nil fields return zero values.
type MockNoteRepository struct {
	CreateNoteFunc        func(domain.Note) (domain.Note, error)
	GetNotesByUserIDFunc  func(userID uint, archived bool) ([]domain.Note, error)
	GetNoteByIDFunc       func(userID, noteID uint) (domain.Note, error)
	UpdateNoteFunc        func(userID, noteID uint, updates map[string]interface{}) error
	DeleteNoteFunc        func(userID, noteID uint) error
	CreateNoteItemFunc    func(userID uint, item domain.NoteItem) (domain.NoteItem, error)
	UpdateNoteItemFunc    func(userID, noteID, itemID uint, updates map[string]interface{}) error
	DeleteNoteItemFunc    func(userID, noteID, itemID uint) error
	ConvertNoteToListFunc func(userID, noteID uint, newType string) error
	ConvertNoteToTextFunc func(userID, noteID uint) error
	ReorderNoteItemsFunc  func(userID, noteID uint, orderedItemIDs []uint) error
}

func (m *MockNoteRepository) CreateNote(n domain.Note) (domain.Note, error) {
	if m.CreateNoteFunc != nil {
		return m.CreateNoteFunc(n)
	}
	return domain.Note{}, nil
}

func (m *MockNoteRepository) GetNotesByUserID(userID uint, archived bool) ([]domain.Note, error) {
	if m.GetNotesByUserIDFunc != nil {
		return m.GetNotesByUserIDFunc(userID, archived)
	}
	return nil, nil
}

func (m *MockNoteRepository) GetNoteByID(userID, noteID uint) (domain.Note, error) {
	if m.GetNoteByIDFunc != nil {
		return m.GetNoteByIDFunc(userID, noteID)
	}
	return domain.Note{}, nil
}

func (m *MockNoteRepository) UpdateNote(userID, noteID uint, updates map[string]interface{}) error {
	if m.UpdateNoteFunc != nil {
		return m.UpdateNoteFunc(userID, noteID, updates)
	}
	return nil
}

func (m *MockNoteRepository) DeleteNote(userID, noteID uint) error {
	if m.DeleteNoteFunc != nil {
		return m.DeleteNoteFunc(userID, noteID)
	}
	return nil
}

func (m *MockNoteRepository) CreateNoteItem(userID uint, item domain.NoteItem) (domain.NoteItem, error) {
	if m.CreateNoteItemFunc != nil {
		return m.CreateNoteItemFunc(userID, item)
	}
	return domain.NoteItem{}, nil
}

func (m *MockNoteRepository) UpdateNoteItem(userID, noteID, itemID uint, updates map[string]interface{}) error {
	if m.UpdateNoteItemFunc != nil {
		return m.UpdateNoteItemFunc(userID, noteID, itemID, updates)
	}
	return nil
}

func (m *MockNoteRepository) DeleteNoteItem(userID, noteID, itemID uint) error {
	if m.DeleteNoteItemFunc != nil {
		return m.DeleteNoteItemFunc(userID, noteID, itemID)
	}
	return nil
}

func (m *MockNoteRepository) ConvertNoteToList(userID, noteID uint, newType string) error {
	if m.ConvertNoteToListFunc != nil {
		return m.ConvertNoteToListFunc(userID, noteID, newType)
	}
	return nil
}

func (m *MockNoteRepository) ConvertNoteToText(userID, noteID uint) error {
	if m.ConvertNoteToTextFunc != nil {
		return m.ConvertNoteToTextFunc(userID, noteID)
	}
	return nil
}

func (m *MockNoteRepository) ReorderNoteItems(userID, noteID uint, orderedItemIDs []uint) error {
	if m.ReorderNoteItemsFunc != nil {
		return m.ReorderNoteItemsFunc(userID, noteID, orderedItemIDs)
	}
	return nil
}
