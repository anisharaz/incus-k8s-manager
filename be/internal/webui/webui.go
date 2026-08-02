// Package webui embeds the built frontend (fe/, built via `pnpm build`) so
// the app can serve it directly. The root Dockerfile builds the frontend
// and copies fe/dist into this package's dist/ directory before compiling.
// Outside Docker, dist/ only contains a placeholder (.gitkeep) — local
// frontend development instead uses Vite's own dev server (see
// fe/vite.config.ts's proxy), so this embed only matters for the
// production image.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS is the built frontend's static file tree, rooted at dist/ (its
// "index.html" is dist/index.html, not the embed's raw "dist/index.html").
var FS = mustSub()

func mustSub() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
