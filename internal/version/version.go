package version

// UnstampedVersion is the value Version holds when the build did not inject a
// release tag, which is the case for a plain `go build`.
const UnstampedVersion = "dev"

var (
	// These values are injected during build time using -ldflags
	Version   = UnstampedVersion
	CommitSHA = "none"
)

// GetVersion returns the current version of sandbox-mcp
func GetVersion() string {
	return Version
}

// GetCommitSHA returns the git commit hash of the build
func GetCommitSHA() string {
	return CommitSHA
}
