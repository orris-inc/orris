package value_objects

import "fmt"

type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionExport Action = "export"
	ActionImport Action = "import"
)

var validActions = map[Action]bool{
	ActionCreate: true,
	ActionRead:   true,
	ActionUpdate: true,
	ActionDelete: true,
	ActionList:   true,
	ActionExport: true,
	ActionImport: true,
}

func NewAction(action string) (Action, error) {
	if action == "" {
		return "", fmt.Errorf("action cannot be empty")
	}

	a := Action(action)
	if !validActions[a] {
		return "", fmt.Errorf("invalid action: %s", action)
	}

	return a, nil
}

func (a Action) String() string {
	return string(a)
}

func (a Action) Equals(other Action) bool {
	return a == other
}
