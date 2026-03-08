package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) GetScratchpad(req domain.GetScratchpadRequest) (domain.GetScratchpadResponse, error) {
	note, err := s.scratchpadRepo.Get(req.UserID, req.Date)
	if err != nil {
		return domain.GetScratchpadResponse{}, err
	}

	return domain.GetScratchpadResponse{
		Content: note.Content,
	}, nil
}

func (s *service) UpsertScratchpad(req domain.UpsertScratchpadRequest) (domain.UpsertScratchpadResponse, error) {
	note := domain.Scratchpad{
		UserId:  req.UserID,
		Date:    req.Date,
		Content: req.Content,
	}

	err := s.scratchpadRepo.Upsert(note)
	if err != nil {
		return domain.UpsertScratchpadResponse{}, err
	}

	return domain.UpsertScratchpadResponse{
		Content: req.Content,
	}, nil
}
