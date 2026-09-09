package repo

// Result is returned by the repo new and goto subcommands. It mirrors the
// spec and plan workflows' result shape, naming the repo being added and the
// folder its code lives at rather than a document path.
type Result struct {
	Step        string `json:"step"`
	RepoPath    string `json:"repo_path"`
	RepoName    string `json:"repo_name"`
	Instruction string `json:"instruction"`
}
