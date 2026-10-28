// Package version holds the build's version.
package version

// Version is stamped at build time (-ldflags -X): the release tag (v0.0.1)
// for release images, dev-<sha> for main builds, "dev" for local builds.
var Version = "dev"
