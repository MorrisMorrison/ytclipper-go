package server

import (
	"fmt"
	"html/template"
	"io/fs"

	"github.com/MorrisMorrison/ytclipper-go/web"

	"github.com/MorrisMorrison/gutils/glogger"
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
		glogger.Log.Infof("Serving web assets from %s", devDir)
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

	e.GET("/", func(c echo.Context) error {
		tmpl, err := homePage()
		if err != nil {
			return err
		}
		return tmpl.Execute(c.Response().Writer, nil)
	})
	return nil
}
