package plan

// Result is returned by the new and goto subcommands.
type Result struct {
	Step        string `json:"step"`
	PlanPath    string `json:"plan_path"`
	PlanName    string `json:"plan_name"`
	Instruction string `json:"instruction"`
}

// StepEntry holds a step name and its current status.
type StepEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// StatusResult is returned by the status subcommand.
type StatusResult struct {
	PlanName       string      `json:"plan_name"`
	PlanPath       string      `json:"plan_path"`
	CurrentStep    string      `json:"current_step"`
	CompletedSteps []string    `json:"completed_steps"`
	TotalSteps     int         `json:"total_steps"`
	Progress       string      `json:"progress"`
	Steps          []StepEntry `json:"steps"`
}

// StepsResult is returned by the steps subcommand.
type StepsResult struct {
	Steps []string `json:"steps"`
}
