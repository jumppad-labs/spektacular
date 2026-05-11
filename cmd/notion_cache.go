package cmd

import (
	"encoding/json"
	"fmt"

	artifactsync "github.com/jumppad-labs/spektacular/internal/artifact/sync"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/spf13/cobra"
)

var notionCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Notion artifact cache sync contracts",
}

var notionCachePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Record Notion MCP content in the local artifact cache",
	RunE:  runNotionCachePull,
}

var notionCacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dirty/stale status for a cached Notion artifact",
	RunE:  runNotionCacheStatus,
}

var notionCachePreparePushCmd = &cobra.Command{
	Use:   "prepare-push",
	Short: "Prepare a safe Notion MCP update or return a merge request",
	RunE:  runNotionCachePreparePush,
}

var notionCacheCommitPushCmd = &cobra.Command{
	Use:   "commit-push",
	Short: "Record returned Notion MCP metadata after a successful update",
	RunE:  runNotionCacheCommitPush,
}

var notionCacheResolveMergeCmd = &cobra.Command{
	Use:   "resolve-merge",
	Short: "Record resolved merge content before retrying a push",
	RunE:  runNotionCacheResolveMerge,
}

func runNotionCachePull(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeCacheSchema(cmd)
	}
	var input artifactsync.PullRequest
	if err := readCacheData(cmd, &input); err != nil {
		return err
	}
	cfg, st, err := cacheConfigAndStore()
	if err != nil {
		return err
	}
	result, err := artifactsync.Pull(st, cfg, input)
	if err != nil {
		return err
	}
	return output.Write(cmd.OutOrStdout(), result, globalFields)
}

func runNotionCacheStatus(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeCacheSchema(cmd)
	}
	var input artifactsync.PushRequest
	if err := readCacheData(cmd, &input); err != nil {
		return err
	}
	cfg, st, err := cacheConfigAndStore()
	if err != nil {
		return err
	}
	result, err := artifactsync.Status(st, cfg, input)
	if err != nil {
		return err
	}
	return output.Write(cmd.OutOrStdout(), result, globalFields)
}

func runNotionCachePreparePush(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeCacheSchema(cmd)
	}
	var input artifactsync.PushRequest
	if err := readCacheData(cmd, &input); err != nil {
		return err
	}
	cfg, st, err := cacheConfigAndStore()
	if err != nil {
		return err
	}
	result, err := artifactsync.PreparePush(st, cfg, input)
	if err != nil {
		return err
	}
	return output.Write(cmd.OutOrStdout(), result, globalFields)
}

func runNotionCacheCommitPush(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeCacheSchema(cmd)
	}
	var input artifactsync.CommitRequest
	if err := readCacheData(cmd, &input); err != nil {
		return err
	}
	cfg, st, err := cacheConfigAndStore()
	if err != nil {
		return err
	}
	result, err := artifactsync.CommitPush(st, cfg, input)
	if err != nil {
		return err
	}
	return output.Write(cmd.OutOrStdout(), result, globalFields)
}

func runNotionCacheResolveMerge(cmd *cobra.Command, _ []string) error {
	if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
		return writeCacheSchema(cmd)
	}
	var input artifactsync.MergeRequestInput
	if err := readCacheData(cmd, &input); err != nil {
		return err
	}
	cfg, st, err := cacheConfigAndStore()
	if err != nil {
		return err
	}
	result, err := artifactsync.ResolveMerge(st, cfg, input)
	if err != nil {
		return err
	}
	return output.Write(cmd.OutOrStdout(), result, globalFields)
}

func readCacheData(cmd *cobra.Command, target any) error {
	dataStr, _ := cmd.Flags().GetString("data")
	if dataStr == "" {
		return fmt.Errorf("--data is required")
	}
	if err := json.Unmarshal([]byte(dataStr), target); err != nil {
		return fmt.Errorf("parsing --data: %w", err)
	}
	return nil
}

func cacheConfigAndStore() (config.Config, store.Store, error) {
	cfg, err := loadConfig()
	if err != nil {
		return config.Config{}, nil, err
	}
	dataDir, err := dataDir()
	if err != nil {
		return config.Config{}, nil, err
	}
	return cfg, store.NewFileStore(dataDir), nil
}

func writeCacheSchema(cmd *cobra.Command) error {
	return output.Write(cmd.OutOrStdout(), commandSchema{
		Input: &schemaObj{
			Type: "object",
			Properties: map[string]*schemaProp{
				"kind":             {Type: "string"},
				"name":             {Type: "string"},
				"content":          {Type: "string"},
				"remote":           {Type: "object"},
				"remote_version":   {Type: "string"},
				"remote_content":   {Type: "string"},
				"resolved_content": {Type: "string"},
				"notion_url":       {Type: "string"},
				"page_id":          {Type: "string"},
				"external_id":      {Type: "string"},
			},
		},
		Output: &schemaObj{
			Type: "object",
			Properties: map[string]*schemaProp{
				"status":        {Type: "string"},
				"entry":         {Type: "object"},
				"local_content": {Type: "string"},
				"merge_request": {Type: "object"},
			},
		},
	}, "")
}

func init() {
	notionCacheCmd.AddCommand(
		notionCachePullCmd,
		notionCacheStatusCmd,
		notionCachePreparePushCmd,
		notionCacheCommitPushCmd,
		notionCacheResolveMergeCmd,
	)
	notionCmd.AddCommand(notionCacheCmd)
}
