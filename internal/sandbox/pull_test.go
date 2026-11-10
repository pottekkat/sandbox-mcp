package sandbox

import (
	"strings"
	"testing"

	"github.com/pottekkat/sandbox-mcp/internal/version"
)

// TestReleaseAssetURL pins the URL that --pull downloads from. A release tag keeps
// its "v" prefix in the asset path, so the tag must not be stripped on the way in.
func TestReleaseAssetURL(t *testing.T) {
	got := releaseAssetURL("v0.1.1")
	want := "https://github.com/pottekkat/sandbox-mcp/releases/download/v0.1.1/sandboxes.tar.gz"
	if got != want {
		t.Fatalf("releaseAssetURL(%q) = %q, want %q", "v0.1.1", got, want)
	}
}

// TestPullSandboxes_UnstampedBuildFailsLoudly covers the reported symptom: a build
// with no injected version used to request .../download/dev/sandboxes.tar.gz and
// report a bare "HTTP 404", which says nothing about the actual cause.
func TestPullSandboxes_UnstampedBuildFailsLoudly(t *testing.T) {
	original := version.Version
	version.Version = version.UnstampedVersion
	t.Cleanup(func() { version.Version = original })

	err := PullSandboxes(t.TempDir(), false)
	if err == nil {
		t.Fatal("expected an error for a build without a release version")
	}
	msg := err.Error()
	if strings.Contains(msg, "404") {
		t.Fatalf("error still reports a bare HTTP status: %q", msg)
	}
	if !strings.Contains(msg, "no release version") {
		t.Fatalf("error does not explain the cause: %q", msg)
	}
}

// TestPullSandboxes_EmptyVersionFailsLoudly guards the other way a version can be
// missing: an ldflags injection that is present but empty.
func TestPullSandboxes_EmptyVersionFailsLoudly(t *testing.T) {
	original := version.Version
	version.Version = ""
	t.Cleanup(func() { version.Version = original })

	if err := PullSandboxes(t.TempDir(), false); err == nil {
		t.Fatal("expected an error for an empty release version")
	}
}
