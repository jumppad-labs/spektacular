package sync

import (
	"fmt"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
)

const (
	StatusPulled        = "pulled"
	StatusReady         = "ready"
	StatusClean         = "clean"
	StatusMergeRequired = "merge_required"
	StatusCommitted     = "committed"
	StatusResolved      = "resolved"
)

// PullRequest records content and remote metadata fetched through Notion MCP.
type PullRequest struct {
	Kind    string                  `json:"kind"`
	Name    string                  `json:"name"`
	Content string                  `json:"content"`
	Remote  artifact.RemoteMetadata `json:"remote"`
}

// IdentityRequest locates an existing cached artifact.
type IdentityRequest struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	NotionURL  string `json:"notion_url"`
	PageID     string `json:"page_id"`
	ExternalID string `json:"external_id"`
}

// PushRequest prepares a local artifact for a Notion MCP update.
type PushRequest struct {
	IdentityRequest
	RemoteVersion string `json:"remote_version"`
	RemoteContent string `json:"remote_content"`
}

// CommitRequest records metadata returned after a successful Notion MCP update.
type CommitRequest struct {
	IdentityRequest
	Remote artifact.RemoteMetadata `json:"remote"`
}

// MergeRequestInput records a completed user/agent merge.
type MergeRequestInput struct {
	IdentityRequest
	Remote          artifact.RemoteMetadata `json:"remote"`
	RemoteContent   string                  `json:"remote_content"`
	ResolvedContent string                  `json:"resolved_content"`
}

// Result is the structured sync contract returned by cache commands.
type Result struct {
	Status       string          `json:"status"`
	Entry        artifact.Entry  `json:"entry,omitempty"`
	LocalPath    string          `json:"local_path,omitempty"`
	LocalContent string          `json:"local_content,omitempty"`
	StatusDetail artifact.Status `json:"status_detail,omitempty"`
	MergeRequest *MergeRequest   `json:"merge_request,omitempty"`
	Changed      bool            `json:"changed,omitempty"`
	Next         string          `json:"next,omitempty"`
}

// MergeRequest contains all content needed to resolve a stale remote safely.
type MergeRequest struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	LocalPath       string `json:"local_path"`
	NotionURL       string `json:"notion_url"`
	PageID          string `json:"page_id"`
	ExternalID      string `json:"external_id"`
	BaselineVersion string `json:"baseline_version"`
	RemoteVersion   string `json:"remote_version"`
	BaselineContent string `json:"baseline_content"`
	LocalContent    string `json:"local_content"`
	RemoteContent   string `json:"remote_content"`
	RequiredAction  string `json:"required_action"`
}

// Pull writes fetched content to the cache and records its manifest baseline.
func Pull(st store.Store, cfg config.Config, req PullRequest) (Result, error) {
	if err := validateRemote(req.Remote); err != nil {
		return Result{}, err
	}
	if req.Kind == "" || req.Name == "" {
		return Result{}, fmt.Errorf("kind and name are required")
	}
	entry, err := artifact.RecordPull(st, cfg, req.Kind, req.Name, []byte(req.Content), req.Remote)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Status:    StatusPulled,
		Entry:     entry,
		LocalPath: entry.LocalPath,
		Changed:   true,
		Next:      "Edit the local cache file, then run prepare-push before updating Notion.",
	}, nil
}

// Status returns the dirty/stale status for a cached artifact.
func Status(st store.Store, cfg config.Config, req PushRequest) (Result, error) {
	entry, err := lookupEntry(st, cfg, req.IdentityRequest)
	if err != nil {
		return Result{}, err
	}
	status, err := artifact.EntryStatus(st, entry, req.RemoteVersion)
	if err != nil {
		return Result{}, err
	}
	return Result{Status: "status", Entry: entry, LocalPath: entry.LocalPath, StatusDetail: status}, nil
}

// PreparePush decides whether a local edit can be safely pushed to Notion.
func PreparePush(st store.Store, cfg config.Config, req PushRequest) (Result, error) {
	entry, err := lookupEntry(st, cfg, req.IdentityRequest)
	if err != nil {
		return Result{}, err
	}
	status, err := artifact.EntryStatus(st, entry, req.RemoteVersion)
	if err != nil {
		return Result{}, err
	}
	localContent, err := st.Read(entry.LocalPath)
	if err != nil {
		return Result{}, err
	}
	if status.Stale {
		baseline, err := artifact.BaselineContent(st, entry)
		if err != nil {
			return Result{}, err
		}
		return Result{
			Status:       StatusMergeRequired,
			Entry:        entry,
			LocalPath:    entry.LocalPath,
			StatusDetail: status,
			MergeRequest: &MergeRequest{
				Kind:            entry.Kind,
				Name:            entry.Name,
				LocalPath:       entry.LocalPath,
				NotionURL:       entry.NotionURL,
				PageID:          entry.PageID,
				ExternalID:      entry.ExternalID,
				BaselineVersion: entry.RemoteVersion,
				RemoteVersion:   req.RemoteVersion,
				BaselineContent: string(baseline),
				LocalContent:    string(localContent),
				RemoteContent:   req.RemoteContent,
				RequiredAction:  "Merge baseline, local, and remote content; then run resolve-merge before retrying prepare-push.",
			},
			Next: "Run resolve-merge with resolved content before updating Notion.",
		}, nil
	}
	if !status.Dirty {
		return Result{
			Status:       StatusClean,
			Entry:        entry,
			LocalPath:    entry.LocalPath,
			StatusDetail: status,
			Next:         "No Notion update is required.",
		}, nil
	}
	return Result{
		Status:       StatusReady,
		Entry:        entry,
		LocalPath:    entry.LocalPath,
		LocalContent: string(localContent),
		StatusDetail: status,
		Changed:      true,
		Next:         "Update the Notion page with local_content, then run commit-push with the returned remote metadata.",
	}, nil
}

// CommitPush records a successful Notion MCP update as the new clean baseline.
func CommitPush(st store.Store, cfg config.Config, req CommitRequest) (Result, error) {
	if err := validateRemote(req.Remote); err != nil {
		return Result{}, err
	}
	entry, err := lookupEntry(st, cfg, req.IdentityRequest)
	if err != nil {
		return Result{}, err
	}
	content, err := st.Read(entry.LocalPath)
	if err != nil {
		return Result{}, err
	}
	entry, err = artifact.UpdateBaseline(st, cfg, entry, content, req.Remote)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Status:    StatusCommitted,
		Entry:     entry,
		LocalPath: entry.LocalPath,
		Changed:   true,
		Next:      "The cache is clean against the returned Notion version.",
	}, nil
}

// ResolveMerge records resolved content and advances the baseline to the remote version.
func ResolveMerge(st store.Store, cfg config.Config, req MergeRequestInput) (Result, error) {
	if err := validateRemote(req.Remote); err != nil {
		return Result{}, err
	}
	entry, err := lookupEntry(st, cfg, req.IdentityRequest)
	if err != nil {
		return Result{}, err
	}
	if req.ResolvedContent == "" {
		return Result{}, fmt.Errorf("resolved_content is required")
	}
	if err := st.Write(entry.LocalPath, []byte(req.ResolvedContent)); err != nil {
		return Result{}, err
	}
	entry, err = artifact.UpdateBaseline(st, cfg, entry, []byte(req.RemoteContent), req.Remote)
	if err != nil {
		return Result{}, err
	}
	status, err := artifact.EntryStatus(st, entry, req.Remote.RemoteVersion)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Status:       StatusResolved,
		Entry:        entry,
		LocalPath:    entry.LocalPath,
		StatusDetail: status,
		Changed:      true,
		Next:         "Retry prepare-push with the same remote version before updating Notion.",
	}, nil
}

func lookupEntry(st store.Store, cfg config.Config, req IdentityRequest) (artifact.Entry, error) {
	manifest, err := artifact.LoadManifest(st, cfg)
	if err != nil {
		return artifact.Entry{}, err
	}
	if req.Kind != "" && req.Name != "" {
		entry, ok := manifest.Entries[artifact.Key(req.Kind, req.Name)]
		if ok {
			return entry, nil
		}
	}
	entry, ok := manifest.Find(artifact.Identity{
		Kind:       req.Kind,
		NotionURL:  req.NotionURL,
		PageID:     req.PageID,
		ExternalID: req.ExternalID,
	})
	if !ok {
		return artifact.Entry{}, fmt.Errorf("cached artifact not found")
	}
	return entry, nil
}

func validateRemote(remote artifact.RemoteMetadata) error {
	return artifact.ValidateRemoteMetadata(remote)
}
