package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) CreateFeedback(req domain.CreateFeedbackRequest) (domain.CreateFeedbackResponse, error) {
	feedback := domain.Feedback{
		UserID:  req.UserID,
		Type:    req.Type,
		AppArea: req.AppArea,
		Message: req.Message,
	}

	resp, err := s.feedbackRepo.CreateFeedback(feedback)
	if err != nil {
		return domain.CreateFeedbackResponse{}, err
	}

	return domain.CreateFeedbackResponse{ID: resp.ID}, nil
}
