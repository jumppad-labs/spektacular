package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
)

const (
	KindSpec     = "spec"
	KindPlan     = "plan"
	KindContext  = "context"
	KindResearch = "research"

	manifestFilename = "manifest.json"
)

// RemoteMetadata is the Notion identity and version data recorded on pull.
type RemoteMetadata struct {
	NotionURL     string `json:"notion_url"`
	PageID        string `json:"page_id"`
	DataSourceURL string `json:"data_source_url"`
	ExternalID    string `json:"external_id"`
	RemoteVersion string `json:"remote_version"`
	Title         string `json:"title,omitempty"`
}

// RemoteMetadataFromAny decodes metadata that may have round-tripped through workflow JSON state.
func RemoteMetadataFromAny(value any) (RemoteMetadata, bool, error) {
	if value == nil {
		return RemoteMetadata{}, false, nil
	}
	switch typed := value.(type) {
	case RemoteMetadata:
		return typed, true, nil
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err != nil {
			return RemoteMetadata{}, true, err
		}
		var remote RemoteMetadata
		if err := json.Unmarshal(raw, &remote); err != nil {
			return RemoteMetadata{}, true, err
		}
		return remote, true, nil
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return RemoteMetadata{}, true, err
		}
		var remote RemoteMetadata
		if err := json.Unmarshal(raw, &remote); err != nil {
			return RemoteMetadata{}, true, err
		}
		return remote, true, nil
	}
}

// ValidateRemoteMetadata checks the normalized Notion MCP metadata fields.
func ValidateRemoteMetadata(remote RemoteMetadata) error {
	switch {
	case remote.NotionURL == "":
		return fmt.Errorf("remote.notion_url is required")
	case remote.PageID == "":
		return fmt.Errorf("remote.page_id is required")
	case remote.DataSourceURL == "":
		return fmt.Errorf("remote.data_source_url is required")
	case remote.ExternalID == "":
		return fmt.Errorf("remote.external_id is required")
	case remote.RemoteVersion == "":
		return fmt.Errorf("remote.remote_version is required")
	default:
		return nil
	}
}

// Entry is one cached artifact tracked by the Notion manifest.
type Entry struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	LocalPath     string `json:"local_path"`
	BaselinePath  string `json:"baseline_path"`
	Checksum      string `json:"checksum"`
	NotionURL     string `json:"notion_url"`
	PageID        string `json:"page_id"`
	DataSourceURL string `json:"data_source_url"`
	ExternalID    string `json:"external_id"`
	RemoteVersion string `json:"remote_version"`
	Title         string `json:"title,omitempty"`
}

// Manifest stores cached artifact metadata for Notion-backed projects.
type Manifest struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
}

// Identity locates a cached artifact using Notion or Spektacular identifiers.
type Identity struct {
	Kind       string
	NotionURL  string
	PageID     string
	ExternalID string
}

// Status reports whether a cached artifact differs from its pulled baseline.
type Status struct {
	Entry           Entry  `json:"entry"`
	Exists          bool   `json:"exists"`
	Dirty           bool   `json:"dirty"`
	Stale           bool   `json:"stale"`
	CurrentChecksum string `json:"current_checksum,omitempty"`
	RemoteVersion   string `json:"remote_version,omitempty"`
}

// Checksum returns the stable SHA-256 checksum for content.
func Checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Path resolves the store-relative path for an artifact in the configured backend.
func Path(cfg config.Config, kind, name string) (string, error) {
	if err := validateKind(kind); err != nil {
		return "", err
	}
	if err := validateName(name); err != nil {
		return "", err
	}

	if cfg.Artifacts.Backend == config.ArtifactBackendNotion {
		return notionPath(cfg.Artifacts.CacheDir, kind, name), nil
	}
	return localPath(kind, name), nil
}

// ManifestPath returns the store-relative Notion manifest path.
func ManifestPath(cfg config.Config) string {
	cacheDir := cfg.Artifacts.CacheDir
	if cacheDir == "" {
		cacheDir = config.DefaultNotionCacheDir
	}
	return filepath.ToSlash(filepath.Join(cacheDir, manifestFilename))
}

// LoadManifest reads the Notion artifact manifest. Missing manifests are empty.
func LoadManifest(st store.Store, cfg config.Config) (Manifest, error) {
	raw, err := st.Read(ManifestPath(cfg))
	if err == store.ErrNotFound {
		return NewManifest(), nil
	}
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parsing artifact manifest: %w", err)
	}
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]Entry{}
	}
	return manifest, nil
}

// SaveManifest writes the Notion artifact manifest.
func SaveManifest(st store.Store, cfg config.Config, manifest Manifest) error {
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]Entry{}
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling artifact manifest: %w", err)
	}
	return st.Write(ManifestPath(cfg), raw)
}

// NewManifest returns an empty manifest.
func NewManifest() Manifest {
	return Manifest{Version: 1, Entries: map[string]Entry{}}
}

// RecordPull writes pulled content to cache and records its remote baseline.
func RecordPull(st store.Store, cfg config.Config, kind, name string, content []byte, remote RemoteMetadata) (Entry, error) {
	localPath, err := Path(cfg, kind, name)
	if err != nil {
		return Entry{}, err
	}
	baselinePath, err := BaselinePath(cfg, kind, name)
	if err != nil {
		return Entry{}, err
	}
	if err := st.Write(localPath, content); err != nil {
		return Entry{}, err
	}
	if err := st.Write(baselinePath, content); err != nil {
		return Entry{}, err
	}

	manifest, err := LoadManifest(st, cfg)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		Kind:          kind,
		Name:          name,
		LocalPath:     localPath,
		BaselinePath:  baselinePath,
		Checksum:      Checksum(content),
		NotionURL:     remote.NotionURL,
		PageID:        remote.PageID,
		DataSourceURL: remote.DataSourceURL,
		ExternalID:    remote.ExternalID,
		RemoteVersion: remote.RemoteVersion,
		Title:         remote.Title,
	}
	manifest.Entries[Key(kind, name)] = entry
	if err := SaveManifest(st, cfg, manifest); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// UpdateBaseline records content as the clean remote baseline for an entry.
func UpdateBaseline(st store.Store, cfg config.Config, entry Entry, content []byte, remote RemoteMetadata) (Entry, error) {
	if entry.BaselinePath == "" {
		baselinePath, err := BaselinePath(cfg, entry.Kind, entry.Name)
		if err != nil {
			return Entry{}, err
		}
		entry.BaselinePath = baselinePath
	}
	if err := st.Write(entry.BaselinePath, content); err != nil {
		return Entry{}, err
	}

	entry.Checksum = Checksum(content)
	entry.NotionURL = firstNonEmpty(remote.NotionURL, entry.NotionURL)
	entry.PageID = firstNonEmpty(remote.PageID, entry.PageID)
	entry.DataSourceURL = firstNonEmpty(remote.DataSourceURL, entry.DataSourceURL)
	entry.ExternalID = firstNonEmpty(remote.ExternalID, entry.ExternalID)
	entry.RemoteVersion = firstNonEmpty(remote.RemoteVersion, entry.RemoteVersion)
	entry.Title = firstNonEmpty(remote.Title, entry.Title)

	manifest, err := LoadManifest(st, cfg)
	if err != nil {
		return Entry{}, err
	}
	manifest.Entries[Key(entry.Kind, entry.Name)] = entry
	if err := SaveManifest(st, cfg, manifest); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// Find returns the first manifest entry matching the supplied identity.
func (m Manifest) Find(identity Identity) (Entry, bool) {
	for _, entry := range m.Entries {
		if identity.Kind != "" && entry.Kind != identity.Kind {
			continue
		}
		switch {
		case identity.NotionURL != "" && entry.NotionURL == identity.NotionURL:
			return entry, true
		case identity.PageID != "" && entry.PageID == identity.PageID:
			return entry, true
		case identity.ExternalID != "" && entry.ExternalID == identity.ExternalID:
			return entry, true
		}
	}
	return Entry{}, false
}

// EntryStatus compares the current cache file against the manifest baseline.
func EntryStatus(st store.Store, entry Entry, currentRemoteVersion string) (Status, error) {
	status := Status{
		Entry:         entry,
		RemoteVersion: currentRemoteVersion,
		Stale:         currentRemoteVersion != "" && currentRemoteVersion != entry.RemoteVersion,
	}
	content, err := st.Read(entry.LocalPath)
	if err == store.ErrNotFound {
		return status, nil
	}
	if err != nil {
		return Status{}, err
	}
	status.Exists = true
	status.CurrentChecksum = Checksum(content)
	status.Dirty = status.CurrentChecksum != entry.Checksum
	return status, nil
}

// BaselineContent reads the pulled remote baseline for an entry.
func BaselineContent(st store.Store, entry Entry) ([]byte, error) {
	if entry.BaselinePath == "" {
		return nil, fmt.Errorf("artifact %s is missing baseline path", Key(entry.Kind, entry.Name))
	}
	content, err := st.Read(entry.BaselinePath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// Key returns the stable manifest key for an artifact.
func Key(kind, name string) string {
	return kind + ":" + name
}

// BaselinePath resolves the store-relative baseline path for an artifact.
func BaselinePath(cfg config.Config, kind, name string) (string, error) {
	if err := validateKind(kind); err != nil {
		return "", err
	}
	if err := validateName(name); err != nil {
		return "", err
	}
	cacheDir := cfg.Artifacts.CacheDir
	if cacheDir == "" {
		cacheDir = config.DefaultNotionCacheDir
	}
	switch kind {
	case KindSpec:
		return filepath.ToSlash(filepath.Join(cacheDir, ".baseline", "specs", name+".md")), nil
	case KindPlan:
		return filepath.ToSlash(filepath.Join(cacheDir, ".baseline", "plans", name, "plan.md")), nil
	case KindContext:
		return filepath.ToSlash(filepath.Join(cacheDir, ".baseline", "plans", name, "context.md")), nil
	case KindResearch:
		return filepath.ToSlash(filepath.Join(cacheDir, ".baseline", "plans", name, "research.md")), nil
	default:
		return "", fmt.Errorf("unsupported artifact kind %q", kind)
	}
}

func notionPath(cacheDir, kind, name string) string {
	if cacheDir == "" {
		cacheDir = config.DefaultNotionCacheDir
	}
	switch kind {
	case KindSpec:
		return filepath.ToSlash(filepath.Join(cacheDir, "specs", name+".md"))
	case KindPlan:
		return filepath.ToSlash(filepath.Join(cacheDir, "plans", name, "plan.md"))
	case KindContext:
		return filepath.ToSlash(filepath.Join(cacheDir, "plans", name, "context.md"))
	case KindResearch:
		return filepath.ToSlash(filepath.Join(cacheDir, "plans", name, "research.md"))
	default:
		return ""
	}
}

func localPath(kind, name string) string {
	switch kind {
	case KindSpec:
		return filepath.ToSlash(filepath.Join("specs", name+".md"))
	case KindPlan:
		return filepath.ToSlash(filepath.Join("plans", name, "plan.md"))
	case KindContext:
		return filepath.ToSlash(filepath.Join("plans", name, "context.md"))
	case KindResearch:
		return filepath.ToSlash(filepath.Join("plans", name, "research.md"))
	default:
		return ""
	}
}

func validateKind(kind string) error {
	switch kind {
	case KindSpec, KindPlan, KindContext, KindResearch:
		return nil
	default:
		return fmt.Errorf("unsupported artifact kind %q", kind)
	}
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("artifact name is required")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("artifact name must not contain path separators")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("artifact name must not be %q", name)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
