package artifact

import (
	"path/filepath"
	"testing"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

func notionConfigForTest() config.Config {
	cfg := config.NewDefault()
	cfg.Spec.IDMethod = config.SpecIDMethodExternal
	cfg.Artifacts.Backend = config.ArtifactBackendNotion
	cfg.Artifacts.Notion.SpecsDataSource = "collection://specs"
	cfg.Artifacts.Notion.PlansDataSource = "collection://plans"
	return cfg
}

func remoteForTest() RemoteMetadata {
	return RemoteMetadata{
		NotionURL:     "https://notion.so/spec",
		PageID:        "page-123",
		DataSourceURL: "collection://specs",
		ExternalID:    "SPEC-1",
		RemoteVersion: "2026-05-09T12:00:00Z",
		Title:         "Billing Export",
	}
}

func TestPath_NotionArtifactsResolveOutsideLocalArtifactDirectories(t *testing.T) {
	cfg := notionConfigForTest()

	specPath, err := Path(cfg, KindSpec, "20260509-billing")
	require.NoError(t, err)
	planPath, err := Path(cfg, KindPlan, "20260509-billing")
	require.NoError(t, err)

	require.Equal(t, "cache/notion/specs/20260509-billing.md", specPath)
	require.Equal(t, "cache/notion/plans/20260509-billing/plan.md", planPath)
	require.NotEqual(t, "specs/20260509-billing.md", specPath)
	require.NotEqual(t, "plans/20260509-billing/plan.md", planPath)
}

func TestPath_LocalArtifactsKeepExistingLocations(t *testing.T) {
	cfg := config.NewDefault()

	specPath, err := Path(cfg, KindSpec, "20260509-billing")
	require.NoError(t, err)
	planPath, err := Path(cfg, KindPlan, "20260509-billing")
	require.NoError(t, err)

	require.Equal(t, "specs/20260509-billing.md", specPath)
	require.Equal(t, "plans/20260509-billing/plan.md", planPath)
}

func TestRecordPullWritesCacheAndManifestMetadata(t *testing.T) {
	cfg := notionConfigForTest()
	st := store.NewFileStore(t.TempDir())
	content := []byte("# Spec\n")

	entry, err := RecordPull(st, cfg, KindSpec, "spec-1", content, remoteForTest())
	require.NoError(t, err)

	require.Equal(t, "cache/notion/specs/spec-1.md", entry.LocalPath)
	require.Equal(t, Checksum(content), entry.Checksum)
	require.Equal(t, "https://notion.so/spec", entry.NotionURL)
	require.Equal(t, "page-123", entry.PageID)
	require.Equal(t, "collection://specs", entry.DataSourceURL)
	require.Equal(t, "SPEC-1", entry.ExternalID)
	require.Equal(t, "2026-05-09T12:00:00Z", entry.RemoteVersion)

	cached, err := st.Read(entry.LocalPath)
	require.NoError(t, err)
	require.Equal(t, content, cached)

	manifest, err := LoadManifest(st, cfg)
	require.NoError(t, err)
	require.Equal(t, entry, manifest.Entries[Key(KindSpec, "spec-1")])
}

func TestManifestFindLocatesByNotionAndExternalIdentity(t *testing.T) {
	manifest := NewManifest()
	entry := Entry{
		Kind:       KindSpec,
		Name:       "spec-1",
		NotionURL:  "https://notion.so/spec",
		PageID:     "page-123",
		ExternalID: "SPEC-1",
	}
	manifest.Entries[Key(entry.Kind, entry.Name)] = entry

	for _, identity := range []Identity{
		{NotionURL: "https://notion.so/spec"},
		{PageID: "page-123"},
		{Kind: KindSpec, ExternalID: "SPEC-1"},
	} {
		got, ok := manifest.Find(identity)
		require.True(t, ok)
		require.Equal(t, entry, got)
	}

	_, ok := manifest.Find(Identity{Kind: KindPlan, ExternalID: "SPEC-1"})
	require.False(t, ok)
}

func TestEntryStatusDetectsDirtyAndStaleWithoutChangingBaseline(t *testing.T) {
	cfg := notionConfigForTest()
	st := store.NewFileStore(t.TempDir())
	entry, err := RecordPull(st, cfg, KindSpec, "spec-1", []byte("baseline"), remoteForTest())
	require.NoError(t, err)
	require.NoError(t, st.Write(entry.LocalPath, []byte("edited")))

	status, err := EntryStatus(st, entry, "2026-05-09T12:30:00Z")
	require.NoError(t, err)

	require.True(t, status.Exists)
	require.True(t, status.Dirty)
	require.True(t, status.Stale)
	require.Equal(t, Checksum([]byte("baseline")), entry.Checksum)
	require.Equal(t, "2026-05-09T12:00:00Z", entry.RemoteVersion)
	require.Equal(t, Checksum([]byte("edited")), status.CurrentChecksum)
}

func TestPathRejectsTraversalNames(t *testing.T) {
	cfg := notionConfigForTest()

	_, err := Path(cfg, KindSpec, "../escape")
	require.Error(t, err)
	require.Contains(t, err.Error(), "path separators")

	_, err = Path(cfg, "unknown", "spec")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported artifact kind")
}

func TestSaveManifestUsesCachePath(t *testing.T) {
	cfg := notionConfigForTest()
	root := t.TempDir()
	st := store.NewFileStore(root)

	manifest := NewManifest()
	manifest.Entries[Key(KindPlan, "plan-1")] = Entry{Kind: KindPlan, Name: "plan-1"}
	require.NoError(t, SaveManifest(st, cfg, manifest))

	require.FileExists(t, filepath.Join(root, "cache", "notion", "manifest.json"))
}
