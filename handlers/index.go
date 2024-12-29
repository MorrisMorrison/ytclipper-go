package handlers

import (
	"html/template"

	"github.com/labstack/echo/v4"
)

func RenderHomePage(c echo.Context) error {
    tmpl := template.Must(template.ParseFiles("templates/index.html"))
    return tmpl.Execute(c.Response().Writer, nil)
}