package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockFeedbackRepository implements repository.FeedbackRepository with function fields.
type MockFeedbackRepository struct {
	CreateFeedbackFunc func(f domain.Feedback) (domain.Feedback, error)
}

func (m *MockFeedbackRepository) CreateFeedback(f domain.Feedback) (domain.Feedback, error) {
	if m.CreateFeedbackFunc != nil {
		return m.CreateFeedbackFunc(f)
	}
	return domain.Feedback{}, nil
}
