// Package version exposes build-time version information.
package version

//nolint:gochecknoglobals // Set via -ldflags at build time.
var (
	// Version of the application, set at build time.
	version = "dev"
	// Commit hash, set at build time.
	commit = "none"
	// Build date, set at build time.
	date = "unknown"
)

// Info holds the build-time version metadata.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get returns the current build-time version information.
func Get() Info {
	return Info{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}
