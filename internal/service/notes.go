package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) CreateNote(req domain.CreateNoteRequest) (domain.CreateNoteResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CreateNoteResponse{}, err
	}

	note := domain.Note{
		UserID: req.UserID,
		Title:  req.Title,
		Type:   req.Type,
		Color:  req.Color,
	}
	if req.Type == domain.NoteTypeText {
		note.Content = req.Content
	}

	created, err := s.noteRepo.CreateNote(note)
	if err != nil {
		return domain.CreateNoteResponse{}, err
	}

	return domain.CreateNoteResponse{
		ID:         created.ID,
		Title:      created.Title,
		Type:       created.Type,
		Content:    created.Content,
		Color:      created.Color,
		IsPinned:   created.IsPinned,
		IsArchived: created.IsArchived,
		Items:      created.Items,
	}, nil
}

func (s *service) GetNotes(req domain.GetNotesRequest) (domain.GetNotesResponse, error) {
	notes, err := s.noteRepo.GetNotesByUserID(req.UserID)
	if err != nil {
		return domain.GetNotesResponse{}, err
	}
	return domain.GetNotesResponse{Notes: notes}, nil
}

func (s *service) UpdateNote(req domain.UpdateNoteRequest) (domain.UpdateNoteResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.UpdateNoteResponse{}, err
	}

	updates := make(map[string]interface{})
	addContent := req.Content != nil

	if req.Type != nil {
		currentNote, err := s.noteRepo.GetNoteByID(req.UserID, req.ID)
		if err != nil {
			return domain.UpdateNoteResponse{}, err
		}
		currentType := currentNote.Type
		newType := *req.Type

		switch {
		case currentType == newType:
			// same type — no conversion, type not added to map
		case currentType == domain.NoteTypeText:
			// text → checklist|bullet
			if err := s.noteRepo.ConvertNoteToList(req.UserID, req.ID, newType); err != nil {
				return domain.UpdateNoteResponse{}, err
			}
			addContent = false
		case newType == domain.NoteTypeText:
			// checklist|bullet → text
			if err := s.noteRepo.ConvertNoteToText(req.UserID, req.ID); err != nil {
				return domain.UpdateNoteResponse{}, err
			}
			// addContent stays true only if req.Content != nil (prefer explicit content)
		default:
			// checklist ↔ bullet
			updates["type"] = newType
		}
	}

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
	}
	if req.IsArchived != nil {
		updates["is_archived"] = *req.IsArchived
	}
	if addContent {
		updates["content"] = *req.Content
	}

	if len(updates) == 0 {
		return domain.UpdateNoteResponse{}, nil
	}

	if err := s.noteRepo.UpdateNote(req.UserID, req.ID, updates); err != nil {
		return domain.UpdateNoteResponse{}, err
	}

	return domain.UpdateNoteResponse{ID: req.ID}, nil
}

func (s *service) DeleteNote(req domain.DeleteNoteRequest) (domain.DeleteNoteResponse, error) {
	if err := s.noteRepo.DeleteNote(req.UserID, req.ID); err != nil {
		return domain.DeleteNoteResponse{}, err
	}
	return domain.DeleteNoteResponse{ID: req.ID}, nil
}

func (s *service) CreateNoteItem(req domain.CreateNoteItemRequest) (domain.CreateNoteItemResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CreateNoteItemResponse{}, err
	}

	item := domain.NoteItem{
		NoteID:   req.NoteID,
		Content:  req.Content,
		Position: req.Position,
	}

	created, err := s.noteRepo.CreateNoteItem(req.UserID, item)
	if err != nil {
		return domain.CreateNoteItemResponse{}, err
	}

	return domain.CreateNoteItemResponse{
		ID:          created.ID,
		NoteID:      created.NoteID,
		Content:     created.Content,
		IsCompleted: created.IsCompleted,
		Position:    created.Position,
	}, nil
}

func (s *service) UpdateNoteItem(req domain.UpdateNoteItemRequest) (domain.UpdateNoteItemResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.UpdateNoteItemResponse{}, err
	}

	updates := make(map[string]interface{})

	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.IsCompleted != nil {
		updates["is_completed"] = *req.IsCompleted
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}

	if len(updates) == 0 {
		return domain.UpdateNoteItemResponse{}, nil
	}

	if err := s.noteRepo.UpdateNoteItem(req.UserID, req.NoteID, req.ID, updates); err != nil {
		return domain.UpdateNoteItemResponse{}, err
	}

	return domain.UpdateNoteItemResponse{ID: req.ID}, nil
}

func (s *service) DeleteNoteItem(req domain.DeleteNoteItemRequest) (domain.DeleteNoteItemResponse, error) {
	if err := s.noteRepo.DeleteNoteItem(req.UserID, req.NoteID, req.ID); err != nil {
		return domain.DeleteNoteItemResponse{}, err
	}
	return domain.DeleteNoteItemResponse{ID: req.ID}, nil
}

func (s *service) ReorderNoteItems(req domain.ReorderNoteItemsRequest) (domain.ReorderNoteItemsResponse, error) {
	if err := s.noteRepo.ReorderNoteItems(req.UserID, req.NoteID, req.ItemIDs); err != nil {
		return domain.ReorderNoteItemsResponse{}, err
	}
	return domain.ReorderNoteItemsResponse{NoteID: req.NoteID}, nil
}
