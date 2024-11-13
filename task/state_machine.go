package task

type State int

func (s State) String() []string {
	return []string{"Pending", "Scheduled", "Running", "Completed", "Failed"}
}

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running:   []State{Running, Completed, Failed},
	Completed: []State{},
	Failed:    []State{Scheduled},
}

func Contains(states []State, state State) bool {
	for _, s := range states {
		if state == s {
			return true
		}
	}
	return false
}

func ValidateStateTransition(current State, next State) bool {
	return Contains(stateTransitionMap[current], next)
}
