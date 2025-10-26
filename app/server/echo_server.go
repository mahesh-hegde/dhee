package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/config"
)

func StartServer(controller *DheeController, conf *config.DheeConfig, host string, port int) {
	e := echo.New()
	e.Renderer = NewTemplateRenderer(conf)

	e.GET("/favicon.ico", func(c echo.Context) error {
		file, err := templateFs.ReadFile("template/favicon.ico")
		if err != nil {
			// Let's not expose internal server errors, a simple 404 is sufficient
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "image/x-icon", file)
	})

	e.GET("/", controller.GetHome)
	e.GET("/scriptures/:scriptureName/excerpts/:path", controller.GetExcerpts)
	e.GET("/scriptures/:scriptureName/hierarchy", controller.GetHierarchy)
	e.GET("/scriptures/:scriptureName/hierarchy/:path", controller.GetHierarchy).Name = "hierarchy"
	e.GET("/scripture-search", controller.SearchScripture)
	e.GET("/dictionaries/:dictionaryName/words/:word", controller.GetDictionaryWord)
	e.GET("/dictionaries/:dictionaryName/search", controller.SearchDictionary)
	e.GET("/dictionaries/:dictionaryName/suggestions", controller.SuggestDictionary)

	addr := fmt.Sprintf("%s:%d", host, port)
	e.Logger.Fatal(e.Start(addr))
}
