package repository

import (
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	GetGroupsByProjectID(userID, projectID uint) ([]domain.ActionGroup, error)
	GetGroupByID(userID, groupID uint) (domain.ActionGroup, error)
	UpdateGroup(userID, groupID uint, updates map[string]interface{}) error
	CreateEmptyGroup(group domain.ActionGroup) (domain.ActionGroup, error)
	CreateGroupFromAction(userID, actionID uint, group domain.ActionGroup, children []domain.Action) (domain.ActionGroup, []domain.Action, error)
	SetActionGroup(userID, actionID uint, groupID *uint) (domain.Action, error)
	Ungroup(userID, groupID uint) error
	ActiveGroupColors(userID, projectID, exceptGroupID uint) ([]string, error)
	GroupIsComplete(userID, groupID uint) (bool, error)
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

// DeleteProject deletes the project and every action and group in it,
// including scheduled actions, in one transaction.
func (r *GormProjectRepository) DeleteProject(userID, projectID uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("project_id = ? AND user_id = ?", projectID, userID).
			Delete(&domain.Action{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().
			Where("project_id = ? AND user_id = ?", projectID, userID).
			Delete(&domain.ActionGroup{}).Error; err != nil {
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
	err := r.Db.Transaction(func(tx *gorm.DB) error {
		if action.GroupID != nil {
			if _, err := groupInProject(tx, action.UserID, *action.GroupID, action.ProjectID); err != nil {
				return err
			}
		}
		return tx.Omit("Project").Create(&action).Error
	})
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
// children are appended, in order, to the end of their stage's column. They
// stay in the original's group, if it had one. All or nothing.
func (r *GormProjectRepository) ReplaceWithActions(userID, actionID uint, children []domain.Action) ([]domain.Action, error) {
	var created []domain.Action
	err := r.Db.Transaction(func(tx *gorm.DB) error {
		original, err := firstAction(tx, userID, actionID)
		if err != nil {
			return err
		}
		created, err = replaceAction(tx, original, children, original.GroupID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func firstAction(tx *gorm.DB, userID, actionID uint) (domain.Action, error) {
	var a domain.Action
	if err := tx.Where("id = ? AND user_id = ?", actionID, userID).First(&a).Error; err != nil {
		return domain.Action{}, gorm.ErrRecordNotFound
	}
	return a, nil
}

// replaceAction deletes original and creates children in its project and in
// groupID, appended to the end of their stage's column.
func replaceAction(tx *gorm.DB, original domain.Action, children []domain.Action, groupID *uint) ([]domain.Action, error) {
	if err := tx.Unscoped().Delete(&original).Error; err != nil {
		return nil, err
	}
	created := make([]domain.Action, 0, len(children))
	positions := map[domain.Stage]int{}
	for _, child := range children {
		child.UserID = original.UserID
		child.ProjectID = original.ProjectID
		child.GroupID = groupID
		pos, ok := positions[child.Stage]
		if !ok {
			var err error
			if pos, err = nextPosition(tx, original.UserID, original.ProjectID, child.Stage); err != nil {
				return nil, err
			}
		}
		child.Position = pos
		positions[child.Stage] = pos + 1
		if err := tx.Omit("Project").Create(&child).Error; err != nil {
			return nil, err
		}
		created = append(created, child)
	}
	return created, nil
}

func (r *GormProjectRepository) DeleteAction(userID, actionID uint) error {
	action, err := firstAction(r.Db, userID, actionID)
	if err != nil {
		return err
	}
	return r.Db.Unscoped().Delete(&action).Error
}

// --- GROUPS ---

func (r *GormProjectRepository) GetGroupsByProjectID(userID, projectID uint) ([]domain.ActionGroup, error) {
	groups := make([]domain.ActionGroup, 0)
	err := r.Db.
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Order("created_at ASC").Order("id ASC").
		Find(&groups).Error
	return groups, err
}

func (r *GormProjectRepository) GetGroupByID(userID, groupID uint) (domain.ActionGroup, error) {
	var group domain.ActionGroup
	if err := r.Db.Where("id = ? AND user_id = ?", groupID, userID).First(&group).Error; err != nil {
		return domain.ActionGroup{}, gorm.ErrRecordNotFound
	}
	return group, nil
}

func (r *GormProjectRepository) UpdateGroup(userID, groupID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.ActionGroup{}).Where("id = ? AND user_id = ?", groupID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// CreateEmptyGroup makes a group with no actions (New group). The caller has
// checked the project is the user's.
func (r *GormProjectRepository) CreateEmptyGroup(group domain.ActionGroup) (domain.ActionGroup, error) {
	err := r.Db.Create(&group).Error
	return group, err
}

// CreateGroupFromAction is a break down that also creates a group, in the
// original's project, for the children. All or nothing.
func (r *GormProjectRepository) CreateGroupFromAction(userID, actionID uint, group domain.ActionGroup, children []domain.Action) (domain.ActionGroup, []domain.Action, error) {
	var created []domain.Action
	err := r.Db.Transaction(func(tx *gorm.DB) error {
		original, err := firstAction(tx, userID, actionID)
		if err != nil {
			return err
		}
		group.UserID = userID
		group.ProjectID = original.ProjectID
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		created, err = replaceAction(tx, original, children, &group.ID)
		return err
	})
	if err != nil {
		return domain.ActionGroup{}, nil, err
	}
	return group, created, nil
}

// SetActionGroup moves an action into groupID (nil = out of its group). The
// group it left stays, even if empty. groupID must be in the action's
// project, else not found.
func (r *GormProjectRepository) SetActionGroup(userID, actionID uint, groupID *uint) (domain.Action, error) {
	var action domain.Action
	err := r.Db.Transaction(func(tx *gorm.DB) error {
		var err error
		if action, err = firstAction(tx, userID, actionID); err != nil {
			return err
		}
		if groupID != nil {
			if _, err := groupInProject(tx, userID, *groupID, action.ProjectID); err != nil {
				return err
			}
		}
		if err := tx.Model(&action).Omit("Project").Update("group_id", groupID).Error; err != nil {
			return err
		}
		action.GroupID = groupID
		return nil
	})
	return action, err
}

// Ungroup removes a group and leaves its actions where they are, ungrouped.
func (r *GormProjectRepository) Ungroup(userID, groupID uint) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		// Delete first: it locks the group row, so a concurrent move into
		// the group waits for us and then finds it gone.
		result := tx.Unscoped().Where("id = ? AND user_id = ?", groupID, userID).Delete(&domain.ActionGroup{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&domain.Action{}).
			Where("group_id = ? AND user_id = ?", groupID, userID).
			Update("group_id", nil).Error
	})
}

// ActiveGroupColors is the colour of each group on the board that holds one:
// active (empty, or with an unfinished action) or pinned. It leaves out
// exceptGroupID (0 leaves none out). Unpinned completed groups have given
// their colour back, so they don't count.
func (r *GormProjectRepository) ActiveGroupColors(userID, projectID, exceptGroupID uint) ([]string, error) {
	colors := make([]string, 0)
	err := r.Db.
		Table("action_groups AS g").
		Joins("LEFT JOIN actions AS a ON a.group_id = g.id AND a.deleted_at IS NULL").
		Where("g.user_id = ? AND g.project_id = ? AND g.id <> ? AND g.deleted_at IS NULL", userID, projectID, exceptGroupID).
		Group("g.id, g.color, g.is_pinned").
		Having("g.is_pinned = ? OR COUNT(a.id) = 0 OR SUM(CASE WHEN a.stage <> ? THEN 1 ELSE 0 END) > 0", true, domain.StageDone).
		Order("g.id ASC").
		Pluck("g.color", &colors).Error
	if err != nil {
		return nil, err
	}
	return colors, nil
}

// GroupIsComplete reports whether the group has actions and all of them are
// done. Completion is never stored; this works it out.
func (r *GormProjectRepository) GroupIsComplete(userID, groupID uint) (bool, error) {
	var counts struct {
		Total      int64
		Unfinished int64
	}
	err := r.Db.Model(&domain.Action{}).
		Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN stage <> ? THEN 1 ELSE 0 END), 0) AS unfinished", domain.StageDone).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Scan(&counts).Error
	if err != nil {
		return false, err
	}
	return counts.Total > 0 && counts.Unfinished == 0, nil
}

// groupInProject locks the group row (FOR UPDATE) so it can't be removed
// before the caller's transaction commits an action into it.
func groupInProject(tx *gorm.DB, userID, groupID, projectID uint) (domain.ActionGroup, error) {
	var group domain.ActionGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ? AND project_id = ?", groupID, userID, projectID).First(&group).Error; err != nil {
		return domain.ActionGroup{}, gorm.ErrRecordNotFound
	}
	return group, nil
}
