package sync

import (
	"testing"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

func testConfig() config.Config {
	cfg := config.NewDefault()
	cfg.Spec.IDMethod = config.SpecIDMethodExternal
	cfg.Artifacts.Backend = config.ArtifactBackendNotion
	cfg.Artifacts.Notion.SpecsDataSource = "collection://specs"
	cfg.Artifacts.Notion.PlansDataSource = "collection://plans"
	return cfg
}

func testRemote(version string) artifact.RemoteMetadata {
	return artifact.RemoteMetadata{
		NotionURL:     "https://notion.so/spec",
		PageID:        "page-123",
		DataSourceURL: "collection://specs",
		ExternalID:    "SPEC-1",
		RemoteVersion: version,
		Title:         "Spec 1",
	}
}

func TestPullWritesCacheAndManifest(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())

	result, err := Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  testRemote("v1"),
	})
	require.NoError(t, err)

	require.Equal(t, StatusPulled, result.Status)
	require.Equal(t, "cache/notion/specs/spec-1.md", result.Entry.LocalPath)
	require.Equal(t, "cache/notion/.baseline/specs/spec-1.md", result.Entry.BaselinePath)
	require.Equal(t, artifact.Checksum([]byte("baseline")), result.Entry.Checksum)

	manifest, err := artifact.LoadManifest(st, cfg)
	require.NoError(t, err)
	require.Equal(t, result.Entry, manifest.Entries[artifact.Key(artifact.KindSpec, "spec-1")])
}

func TestPreparePushReturnsMergeRequestWhenRemoteChanged(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())
	pulled, err := Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  testRemote("v1"),
	})
	require.NoError(t, err)
	require.NoError(t, st.Write(pulled.Entry.LocalPath, []byte("local edit")))

	result, err := PreparePush(st, cfg, PushRequest{
		IdentityRequest: IdentityRequest{Kind: artifact.KindSpec, Name: "spec-1"},
		RemoteVersion:   "v2",
		RemoteContent:   "remote edit",
	})
	require.NoError(t, err)

	require.Equal(t, StatusMergeRequired, result.Status)
	require.NotNil(t, result.MergeRequest)
	require.Equal(t, "baseline", result.MergeRequest.BaselineContent)
	require.Equal(t, "local edit", result.MergeRequest.LocalContent)
	require.Equal(t, "remote edit", result.MergeRequest.RemoteContent)
	require.Equal(t, "v1", result.MergeRequest.BaselineVersion)
	require.Equal(t, "v2", result.MergeRequest.RemoteVersion)
	require.Contains(t, result.MergeRequest.RequiredAction, "resolve-merge")
}

func TestResolveMergeAllowsRetryingPreparePush(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())
	pulled, err := Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  testRemote("v1"),
	})
	require.NoError(t, err)
	require.NoError(t, st.Write(pulled.Entry.LocalPath, []byte("local edit")))

	resolved, err := ResolveMerge(st, cfg, MergeRequestInput{
		IdentityRequest: IdentityRequest{Kind: artifact.KindSpec, Name: "spec-1"},
		Remote:          testRemote("v2"),
		RemoteContent:   "remote edit",
		ResolvedContent: "merged content",
	})
	require.NoError(t, err)
	require.Equal(t, StatusResolved, resolved.Status)
	require.True(t, resolved.StatusDetail.Dirty)
	require.False(t, resolved.StatusDetail.Stale)

	prepared, err := PreparePush(st, cfg, PushRequest{
		IdentityRequest: IdentityRequest{Kind: artifact.KindSpec, Name: "spec-1"},
		RemoteVersion:   "v2",
		RemoteContent:   "remote edit",
	})
	require.NoError(t, err)
	require.Equal(t, StatusReady, prepared.Status)
	require.Equal(t, "merged content", prepared.LocalContent)
}

func TestCommitPushUpdatesManifestToNewRemoteVersion(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())
	pulled, err := Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  testRemote("v1"),
	})
	require.NoError(t, err)
	require.NoError(t, st.Write(pulled.Entry.LocalPath, []byte("local edit")))

	committed, err := CommitPush(st, cfg, CommitRequest{
		IdentityRequest: IdentityRequest{Kind: artifact.KindSpec, Name: "spec-1"},
		Remote:          testRemote("v2"),
	})
	require.NoError(t, err)

	require.Equal(t, StatusCommitted, committed.Status)
	require.Equal(t, "v2", committed.Entry.RemoteVersion)
	require.Equal(t, artifact.Checksum([]byte("local edit")), committed.Entry.Checksum)
	status, err := artifact.EntryStatus(st, committed.Entry, "v2")
	require.NoError(t, err)
	require.False(t, status.Dirty)
	require.False(t, status.Stale)
}

func TestLookupByNotionURLPageIDAndExternalID(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())
	_, err := Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  testRemote("v1"),
	})
	require.NoError(t, err)

	for _, req := range []IdentityRequest{
		{NotionURL: "https://notion.so/spec"},
		{PageID: "page-123"},
		{Kind: artifact.KindSpec, ExternalID: "SPEC-1"},
	} {
		result, err := Status(st, cfg, PushRequest{IdentityRequest: req})
		require.NoError(t, err)
		require.Equal(t, "spec-1", result.Entry.Name)
	}
}

func TestMissingManifestEntryAndMalformedRemoteReturnErrors(t *testing.T) {
	cfg := testConfig()
	st := store.NewFileStore(t.TempDir())

	_, err := PreparePush(st, cfg, PushRequest{IdentityRequest: IdentityRequest{Kind: artifact.KindSpec, Name: "missing"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	_, err = Pull(st, cfg, PullRequest{
		Kind:    artifact.KindSpec,
		Name:    "spec-1",
		Content: "baseline",
		Remote:  artifact.RemoteMetadata{PageID: "page-123"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "remote.notion_url")
}
