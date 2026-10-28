package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"

	"github.com/MorrisMorrison/ytclipper-go/internal/videoprocessing"
	"github.com/MorrisMorrison/ytclipper-go/web"

	"github.com/labstack/echo/v4"
)

const indexTemplate = "templates/index.html"

// registerWeb serves the page at / and the static files under /static, from
// the assets embedded in the binary or, with devDir set, from disk.
func registerWeb(e *echo.Echo, devDir string) error {
	assets := web.Assets(devDir)
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return fmt.Errorf("static files: %w", err)
	}
	e.StaticFS("/static", static)

	var homePage func() (*template.Template, error)
	if devDir != "" {
		// Parsed on every request, so template edits show up immediately.
		slog.Info("Serving web assets from disk", "dir", devDir)
		homePage = func() (*template.Template, error) {
			return template.ParseFS(assets, indexTemplate)
		}
	} else {
		tmpl, err := template.ParseFS(assets, indexTemplate)
		if err != nil {
			return fmt.Errorf("page template: %w", err)
		}
		homePage = func() (*template.Template, error) { return tmpl, nil }
	}

	page := func(c echo.Context) error {
		tmpl, err := homePage()
		if err != nil {
			return err
		}
		return tmpl.Execute(c.Response().Writer, nil)
	}
	e.GET("/", page)

	// Domain swap: a YouTube link with its host replaced by ours opens the
	// page, which reads the video from its own address. Echo prefers the
	// static routes (/health, /api/…, /static/…) over these.
	e.GET("/watch", videoPage(page, func(c echo.Context) string { return c.QueryParam("v") }))
	e.GET("/shorts/:id", videoPage(page, idParam))
	e.GET("/live/:id", videoPage(page, idParam))
	e.GET("/:id", videoPage(page, idParam)) // youtu.be/ID
	return nil
}

func idParam(c echo.Context) string { return c.Param("id") }

// videoPage serves page when videoID(c) is a valid YouTube video ID, and 404
// otherwise, so unrelated paths keep failing as before.
func videoPage(page echo.HandlerFunc, videoID func(echo.Context) string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !videoprocessing.IsValidVideoID(videoID(c)) {
			return echo.ErrNotFound
		}
		return page(c)
	}
}
