package repository

import (
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

// ProjectRepository owns both projects and their actions, the way
// RoutineRepository owns routines and routine instances.
type ProjectRepository interface {
	CreateProject(domain.Project) (domain.Project, error)
	GetProjectsByUserID(userID uint, archived bool) ([]domain.ProjectWithStats, error)
	GetProjectByID(userID, projectID uint) (domain.Project, error)
	UpdateProject(userID, projectID uint, updates map[string]interface{}) error
	DeleteProject(userID, projectID uint) error

	CreateAction(domain.Action) (domain.Action, error)
	GetActionByID(userID, actionID uint) (domain.Action, error)
	GetActionsByProjectID(userID, projectID uint) ([]domain.Action, error)
	GetActionsInRange(userID uint, start, end time.Time) ([]domain.Action, error)
	GetReadyActions(userID uint) ([]domain.Action, error)
	UpdateAction(userID, actionID uint, updates map[string]interface{}) error
	NextPosition(userID, projectID uint, stage domain.Stage) (int, error)
	FirstPosition(userID, projectID uint, stage domain.Stage) (int, error)
	ReorderActions(userID, projectID uint, stage domain.Stage, orderedIDs []uint) error
	ReplaceWithActions(userID, actionID uint, children []domain.Action) ([]domain.Action, error)
	DeleteAction(userID, actionID uint) error
}

type GormProjectRepository struct {
	Db *gorm.DB
}

// --- PROJECTS ---

func (r *GormProjectRepository) CreateProject(project domain.Project) (domain.Project, error) {
	err := r.Db.Create(&project).Error
	return project, err
}

func (r *GormProjectRepository) GetProjectsByUserID(userID uint, archived bool) ([]domain.ProjectWithStats, error) {
	projects := make([]domain.ProjectWithStats, 0)
	err := r.Db.
		Table("projects AS p").
		Select(`
			p.*,
			COUNT(a.id) AS action_count,
			COALESCE(SUM(CASE WHEN a.stage = ? THEN 1 ELSE 0 END), 0) AS done_count
		`, domain.StageDone).
		Joins("LEFT JOIN actions AS a ON a.project_id = p.id AND a.deleted_at IS NULL").
		Where("p.user_id = ? AND p.deleted_at IS NULL AND p.is_archived = ?", userID, archived).
		Group("p.id").
		Order("p.created_at ASC").
		Scan(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *GormProjectRepository) GetProjectByID(userID, projectID uint) (domain.Project, error) {
	var project domain.Project
	if err := r.Db.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
		return domain.Project{}, gorm.ErrRecordNotFound
	}
	return project, nil
}

func (r *GormProjectRepository) UpdateProject(userID, projectID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.Project{}).Where("id = ? AND user_id = ?", projectID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// DeleteProject deletes the project and every action in it, including
// scheduled ones, in one transaction.
func (r *GormProjectRepository) DeleteProject(userID, projectID uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("project_id = ? AND user_id = ?", projectID, userID).
			Delete(&domain.Action{}).Error; err != nil {
			return err
		}
		result := tx.Unscoped().Where("id = ? AND user_id = ?", projectID, userID).Delete(&domain.Project{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// --- ACTIONS ---

func (r *GormProjectRepository) CreateAction(action domain.Action) (domain.Action, error) {
	if _, err := r.GetProjectByID(action.UserID, action.ProjectID); err != nil {
		return domain.Action{}, gorm.ErrRecordNotFound
	}
	err := r.Db.Omit("Project").Create(&action).Error
	return action, err
}

func (r *GormProjectRepository) GetActionByID(userID, actionID uint) (domain.Action, error) {
	var action domain.Action
	if err := r.Db.Where("id = ? AND user_id = ?", actionID, userID).First(&action).Error; err != nil {
		return domain.Action{}, gorm.ErrRecordNotFound
	}
	return action, nil
}

// GetActionsByProjectID returns a project's whole board. Board stages have no
// date, so they sort by position; scheduled and done sort by date.
func (r *GormProjectRepository) GetActionsByProjectID(userID, projectID uint) ([]domain.Action, error) {
	if _, err := r.GetProjectByID(userID, projectID); err != nil {
		return nil, err
	}
	actions := make([]domain.Action, 0)
	err := r.Db.
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Order("date ASC").Order("position ASC").Order("updated_at ASC").Order("id ASC").
		Find(&actions).Error
	return actions, err
}

// GetActionsInRange returns dated actions (scheduled, or done after being
// scheduled) whose start falls in [start, end], including archived projects'.
func (r *GormProjectRepository) GetActionsInRange(userID uint, start, end time.Time) ([]domain.Action, error) {
	actions := make([]domain.Action, 0)
	err := r.Db.Preload("Project").
		Where("user_id = ? AND date >= ? AND date <= ?", userID, start, end).
		Order("date ASC").
		Find(&actions).Error
	return actions, err
}

// GetReadyActions returns every ready action across the user's active
// (non-archived) projects, for the Plan sidebar.
func (r *GormProjectRepository) GetReadyActions(userID uint) ([]domain.Action, error) {
	actions := make([]domain.Action, 0)
	err := r.Db.Preload("Project").
		Joins("JOIN projects ON projects.id = actions.project_id AND projects.deleted_at IS NULL").
		Where("actions.user_id = ? AND actions.stage = ? AND projects.is_archived = ?", userID, domain.StageReady, false).
		Order("actions.project_id ASC").Order("actions.position ASC").
		Find(&actions).Error
	return actions, err
}

func (r *GormProjectRepository) UpdateAction(userID, actionID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.Action{}).Where("id = ? AND user_id = ?", actionID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// NextPosition is the position that appends to the end of a stage's column.
func (r *GormProjectRepository) NextPosition(userID, projectID uint, stage domain.Stage) (int, error) {
	return nextPosition(r.Db, userID, projectID, stage)
}

func nextPosition(db *gorm.DB, userID, projectID uint, stage domain.Stage) (int, error) {
	var next int
	err := db.Model(&domain.Action{}).
		Select("COALESCE(MAX(position), -1) + 1").
		Where("user_id = ? AND project_id = ? AND stage = ?", userID, projectID, stage).
		Scan(&next).Error
	return next, err
}

// FirstPosition is the position that puts an action at the top of a stage's
// column. It can go below zero.
func (r *GormProjectRepository) FirstPosition(userID, projectID uint, stage domain.Stage) (int, error) {
	var first int
	err := r.Db.Model(&domain.Action{}).
		Select("COALESCE(MIN(position), 1) - 1").
		Where("user_id = ? AND project_id = ? AND stage = ?", userID, projectID, stage).
		Scan(&first).Error
	return first, err
}

func (r *GormProjectRepository) ReorderActions(userID, projectID uint, stage domain.Stage, orderedIDs []uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		var project domain.Project
		if err := tx.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		// Unlike note items, the whole column must be sent: a partial list
		// would leave the unlisted actions sharing positions with listed ones.
		var columnSize, matched int64
		if err := tx.Model(&domain.Action{}).
			Where("project_id = ? AND stage = ?", projectID, stage).
			Count(&columnSize).Error; err != nil {
			return err
		}
		if len(orderedIDs) > 0 {
			if err := tx.Model(&domain.Action{}).
				Where("id IN ? AND project_id = ? AND stage = ?", orderedIDs, projectID, stage).
				Count(&matched).Error; err != nil {
				return err
			}
		}
		if matched != int64(len(orderedIDs)) {
			return fmt.Errorf("%w: some action IDs do not belong to this project and stage", domain.ErrInvalidInput)
		}
		if matched != columnSize {
			return fmt.Errorf("%w: orderedIds must list every action in %s", domain.ErrInvalidInput, stage)
		}
		for i, id := range orderedIDs {
			if err := tx.Model(&domain.Action{}).Where("id = ?", id).Update("position", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceWithActions breaks an action down: the original is deleted and the
// children are appended, in order, to the end of their stage's column. All or
// nothing.
func (r *GormProjectRepository) ReplaceWithActions(userID, actionID uint, children []domain.Action) ([]domain.Action, error) {
	created := make([]domain.Action, 0, len(children))
	err := r.Db.Transaction(func(tx *gorm.DB) error {
		var original domain.Action
		if err := tx.Where("id = ? AND user_id = ?", actionID, userID).First(&original).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Unscoped().Delete(&original).Error; err != nil {
			return err
		}
		positions := map[domain.Stage]int{}
		for _, child := range children {
			child.UserID = userID
			child.ProjectID = original.ProjectID
			pos, ok := positions[child.Stage]
			if !ok {
				var err error
				if pos, err = nextPosition(tx, userID, original.ProjectID, child.Stage); err != nil {
					return err
				}
			}
			child.Position = pos
			positions[child.Stage] = pos + 1
			if err := tx.Omit("Project").Create(&child).Error; err != nil {
				return err
			}
			created = append(created, child)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *GormProjectRepository) DeleteAction(userID, actionID uint) error {
	result := r.Db.Unscoped().Where("id = ? AND user_id = ?", actionID, userID).Delete(&domain.Action{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
