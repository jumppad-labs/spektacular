package knowledge

// Tier declares how a category's entries are retrieved. A category's tier is
// stated once, here in the registry, and every behaviour that depends on it —
// project scaffolding, search exclusion, and the always-applied reader — reads
// it from here rather than restating a category name. Re-tiering a category is
// therefore a single-field change.
type Tier string

const (
	// TierAlwaysApplied marks a category whose entries are loaded in full on
	// every task and are deliberately excluded from search, so the same content
	// is never surfaced twice. Keep these categories small: their whole content
	// is paid for on every task.
	TierAlwaysApplied Tier = "always-applied"
	// TierLookedUp marks a category whose entries are retrieved only when a
	// query matches them. This is the larger reference body of the knowledge
	// base.
	TierLookedUp Tier = "looked-up"
)

// Category is the registry record describing one knowledge category. It is the
// category's contract with both contributors and the assistant: what the
// category is for, what looks similar but belongs elsewhere, how its entries
// are retrieved, and the shape an entry should take. The registry is the single
// source of truth for the category model — project init scaffolds directories
// and READMEs from it, the knowledge Set derives tier behaviour from it, and the
// `knowledge categories` command projects it to JSON for the contribution flow.
type Category struct {
	// Name is the directory name and path prefix for the category, e.g. "glossary".
	Name string `json:"name"`
	// Purpose states what the category is for.
	Purpose string `json:"purpose"`
	// Boundary states what looks similar but belongs in another category, so a
	// contributor can tell categories apart at the edges.
	Boundary string `json:"boundary"`
	// Tier declares how the category's entries are retrieved.
	Tier Tier `json:"tier"`
	// EntryShape describes the shape an entry should take, e.g. "a term and a
	// short gloss".
	EntryShape string `json:"entryShape"`
}

// Categories is the canonical, ordered list of knowledge categories. It is the
// single declaration of the category model: adding, removing, or re-tiering a
// category is a change to this list alone.
var Categories = []Category{
	{
		Name:       "conventions",
		Purpose:    "The rules a team always wants honoured — coding standards, naming schemes, formatting, required patterns, and the house style that every change must follow.",
		Boundary:   "Not the reasoning behind a rule (that is a decision) and not a one-off lesson learned in passing (that is a learning). A convention is a standing rule, stated as an instruction to follow.",
		Tier:       TierAlwaysApplied,
		EntryShape: "An imperative rule with, where useful, a brief note on its scope — short enough to apply without re-reading.",
	},
	{
		Name:       "glossary",
		Purpose:    "The shared vocabulary of the project — the domain and project-specific terms a contributor must understand to read the rest of the knowledge base and the code.",
		Boundary:   "Not an explanation of how a thing works (that is architecture) and not the rationale for a choice (that is a decision). A glossary entry defines what a term means, nothing more.",
		Tier:       TierAlwaysApplied,
		EntryShape: "A term and a short gloss — one or two sentences. Anything longer belongs in architecture, decisions, or learnings.",
	},
	{
		Name:       "architecture",
		Purpose:    "How the system is built and fits together — components and their responsibilities, the boundaries between them, data and control flow, and the structural facts a contributor needs to navigate the code.",
		Boundary:   "Not why a structure was chosen over the alternatives (that is a decision) and not a defined term (that is a glossary entry). Architecture describes what exists and how it works.",
		Tier:       TierLookedUp,
		EntryShape: "A focused description of one component, boundary, or flow, written so a reader can place it in the larger system.",
	},
	{
		Name:       "gotchas",
		Purpose:    "Sharp edges and non-obvious traps — surprising behaviours, easy-to-make mistakes, and the things that bite a contributor who does not already know about them.",
		Boundary:   "Not a standing rule (that is a convention) and not the structure of the system (that is architecture). A gotcha is a warning about a specific trap and how to avoid it.",
		Tier:       TierLookedUp,
		EntryShape: "A short warning naming the trap, why it surprises, and what to do instead.",
	},
	{
		Name:       "learnings",
		Purpose:    "Empirical knowledge gained from doing the work — what was tried, what worked, what did not, and the practical findings that save the next contributor from repeating the effort.",
		Boundary:   "Not a recorded decision with its rationale (that is a decision) and not a standing rule the team must follow (that is a convention). A learning is an observation from experience.",
		Tier:       TierLookedUp,
		EntryShape: "A finding stated plainly, with enough context to know when it applies.",
	},
	{
		Name:       "decisions",
		Purpose:    "The reasoning behind choices the project has made — the options considered, the trade-offs weighed, and why one path was taken over the others (ADR-style).",
		Boundary:   "Not a description of the resulting structure (that is architecture) and not a rule to follow (that is a convention). A decision records the why, not the what or the how.",
		Tier:       TierLookedUp,
		EntryShape: "A record of the decision, the alternatives considered, and the rationale for the choice made.",
	},
}

// AlwaysApplied returns the names of the categories in the always-applied tier,
// in registry order. Both the search-exclusion in the knowledge Set and the
// always-applied reader consult this single derivation, so the two behaviours
// can never drift apart.
func AlwaysApplied() []string {
	var names []string
	for _, c := range Categories {
		if c.Tier == TierAlwaysApplied {
			names = append(names, c.Name)
		}
	}
	return names
}

// CategoryByName returns the category with the given name and true, or a zero
// Category and false if no category in the registry has that name.
func CategoryByName(name string) (Category, bool) {
	for _, c := range Categories {
		if c.Name == name {
			return c, true
		}
	}
	return Category{}, false
}
