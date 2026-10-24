// Package web holds the page (templates/) and its static files (static/),
// embedded into the binary so it does not depend on the working directory.
package web

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed templates static
var embedded embed.FS

// Assets returns the web assets, with templates/ and static/ at the root: the
// copy embedded in the binary, or dir on disk when dir is set (for live
// editing, point it at the repository's web/ directory).
func Assets(dir string) fs.FS {
	if dir != "" {
		return os.DirFS(dir)
	}
	return embedded
}
