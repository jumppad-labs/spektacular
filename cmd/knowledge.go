package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jumppad-labs/spektacular/internal/knowledge"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Search, read, list, and write across configured knowledge sources",
}

var knowledgeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search every configured knowledge source for a keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  runKnowledgeSearch,
}

var knowledgeReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read a knowledge entry from a named scope",
	RunE:  runKnowledgeRead,
}

var knowledgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every knowledge entry across all configured scopes",
	RunE:  runKnowledgeList,
}

var knowledgeWriteCmd = &cobra.Command{
	Use:   "write",
	Short: "Write a knowledge entry into a named scope",
	RunE:  runKnowledgeWrite,
}

var knowledgeSourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List the configured knowledge scopes and their locations",
	RunE:  runKnowledgeSources,
}

var knowledgeSearchOutputSchema = &schemaObj{
	Type:       "object",
	Properties: map[string]*schemaProp{"hits": {Type: "array"}},
}

var knowledgeReadOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"scope":   {Type: "string"},
		"path":    {Type: "string"},
		"content": {Type: "string"},
	},
}

var knowledgeListOutputSchema = &schemaObj{
	Type:       "object",
	Properties: map[string]*schemaProp{"entries": {Type: "array"}},
}

var knowledgeWriteOutputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"scope": {Type: "string"},
		"path":  {Type: "string"},
	},
}

var knowledgeSourcesOutputSchema = &schemaObj{
	Type:       "object",
	Properties: map[string]*schemaProp{"sources": {Type: "array"}},
}

var knowledgeScopePathInputSchema = &schemaObj{
	Type: "object",
	Properties: map[string]*schemaProp{
		"scope": {Type: "string"},
		"path":  {Type: "string"},
	},
	Required: []string{"scope", "path"},
}

// newKnowledgeSet builds a knowledge.Set from the project configuration,
// resolving relative source locations against the working directory.
func newKnowledgeSet() (*knowledge.Set, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	return knowledge.NewSet(cfg, cwd)
}

func runKnowledgeSearch(cmd *cobra.Command, args []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: nil, Output: knowledgeSearchOutputSchema}, "")
	}
	set, err := newKnowledgeSet()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	hits, err := set.Search(args[0])
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	if hits == nil {
		hits = []store.Hit{}
	}
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"hits": hits})
}

func runKnowledgeRead(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: knowledgeScopePathInputSchema, Output: knowledgeReadOutputSchema}, "")
	}
	input, err := knowledgeScopePathData(cmd)
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	set, err := newKnowledgeSet()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	content, err := set.Read(input.Scope, input.Path)
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{
		"scope":   input.Scope,
		"path":    input.Path,
		"content": string(content),
	})
}

func runKnowledgeList(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: nil, Output: knowledgeListOutputSchema}, "")
	}
	set, err := newKnowledgeSet()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	entries, err := set.List()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	if entries == nil {
		entries = []knowledge.Entry{}
	}
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"entries": entries})
}

func runKnowledgeWrite(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: knowledgeScopePathInputSchema, Output: knowledgeWriteOutputSchema}, "")
	}
	input, err := knowledgeScopePathData(cmd)
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	content, err := readKnowledgeContent(cmd)
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	set, err := newKnowledgeSet()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	if err := set.Write(input.Scope, input.Path, content); err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"scope": input.Scope, "path": input.Path})
}

func runKnowledgeSources(cmd *cobra.Command, _ []string) error {
	if schema, _ := cmd.Flags().GetBool("schema"); schema {
		return output.Write(cmd.OutOrStdout(), commandSchema{Input: nil, Output: knowledgeSourcesOutputSchema}, "")
	}
	set, err := newKnowledgeSet()
	if err != nil {
		return output.WriteError(cmd.ErrOrStderr(), err)
	}
	out := output.New(cmd.OutOrStdout(), globalFields)
	return out.WriteResult(map[string]any{"sources": set.Sources()})
}

// knowledgeScopePathInput is the --data payload for the read and write commands.
type knowledgeScopePathInput struct {
	Scope string `json:"scope"`
	Path  string `json:"path"`
}

// knowledgeScopePathData parses and validates the --data flag shared by the
// read and write subcommands.
func knowledgeScopePathData(cmd *cobra.Command) (knowledgeScopePathInput, error) {
	dataStr, _ := cmd.Flags().GetString("data")
	if dataStr == "" {
		return knowledgeScopePathInput{}, fmt.Errorf(`--data is required (e.g. --data '{"scope":"project","path":"learnings/x.md"}')`)
	}
	var input knowledgeScopePathInput
	if err := json.Unmarshal([]byte(dataStr), &input); err != nil {
		return knowledgeScopePathInput{}, fmt.Errorf("parsing --data: %w", err)
	}
	if input.Scope == "" || input.Path == "" {
		return knowledgeScopePathInput{}, fmt.Errorf(`--data must include non-empty "scope" and "path"`)
	}
	return input, nil
}

// readKnowledgeContent reads the entry body for a write: from the --file path
// when set, otherwise from standard input.
func readKnowledgeContent(cmd *cobra.Command) ([]byte, error) {
	filePath, _ := cmd.Flags().GetString("file")
	if filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", filePath, err)
		}
		return content, nil
	}
	content, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	return content, nil
}

func init() {
	knowledgeCmd.PersistentFlags().Bool("schema", false, "Print the input/output schema for this subcommand and exit")

	knowledgeReadCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"scope":"project","path":"learnings/x.md"}')`)
	knowledgeWriteCmd.Flags().StringP("data", "d", "", `JSON input (e.g. '{"scope":"project","path":"learnings/x.md"}')`)
	knowledgeWriteCmd.Flags().String("file", "", "Read entry content from the file at <path> (relative to cwd); stdin is used when omitted")

	knowledgeCmd.AddCommand(knowledgeSearchCmd, knowledgeReadCmd, knowledgeListCmd, knowledgeWriteCmd, knowledgeSourcesCmd)
}
