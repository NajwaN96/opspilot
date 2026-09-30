package buildmeta

import "regexp"

// Values may be overwritten with -ldflags. They are public build labels, not credentials.
var (
	Version = "0.3.0"
	Commit  = "unknown"
	BuiltAt = "unknown"
)

var (
	commitRE  = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	timeRE    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]+Z$`)
	versionRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,40}$`)
)

// Public returns the only release fields the API may show.
// Environment is limited to the two OpsPilot venues. Extra process state is ignored.
func Public(environment string) map[string]string {
	if environment != "local-live" && environment != "aws-portfolio-demo" {
		environment = "local-live"
	}
	commit := Commit
	if !commitRE.MatchString(commit) {
		commit = "unknown"
	}
	built := BuiltAt
	if !timeRE.MatchString(built) {
		built = "unknown"
	}
	version := Version
	if !versionRE.MatchString(version) {
		version = "dev"
	}
	return map[string]string{
		"version":     version,
		"gitCommit":   commit,
		"buildTime":   built,
		"environment": environment,
	}
}
