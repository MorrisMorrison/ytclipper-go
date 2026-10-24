package api

import (
	"html/template"
	"sync"

	"github.com/labstack/echo/v4"
)

var homePage = sync.OnceValue(func() *template.Template {
	return template.Must(template.ParseFiles("templates/index.html"))
})

func RenderHomePage(c echo.Context) error {
	return homePage().Execute(c.Response().Writer, nil)
}
