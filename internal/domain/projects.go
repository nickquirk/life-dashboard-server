package domain

// Stage is which board column an Action is in. It is the single source of
// truth for an Action's lifecycle — there is no separate status flag.
type Stage string

const (
	StageBraindump Stage = "braindump"
	StageRefine    Stage = "refine"
	StageReady     Stage = "ready"
	StageDone      Stage = "done"
)

func (s Stage) Valid() bool {
	switch s {
	case StageBraindump, StageRefine, StageReady, StageDone:
		return true
	}
	return false
}

// IsBoardStage reports whether the user can move an Action into s by hand.
// Only done is reached by its own verb.
func (s Stage) IsBoardStage() bool {
	switch s {
	case StageBraindump, StageRefine, StageReady:
		return true
	}
	return false
}
