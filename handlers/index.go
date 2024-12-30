package handlers

import (
	"html/template"

	"github.com/labstack/echo/v4"
)

func RenderHomePage(c echo.Context) error {
    homePage := template.Must(template.ParseFiles("templates/index.html"))
    return homePage.Execute(c.Response().Writer, nil)
}