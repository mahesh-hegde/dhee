package server

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/config"
)

func StartServer(controller *DheeController, conf *config.DheeConfig, host string, port int) {
	e := echo.New()
	e.Renderer = NewTemplateRenderer()

	e.GET("/", controller.GetHome)
	e.GET("/scriptures/:scriptureName/excerpts/:path", controller.GetExcerpts)
	e.GET("/scripture-search", controller.SearchScripture)
	e.GET("/dictionaries/:dictionaryName/words/:word", controller.GetDictionaryWord)
	e.GET("/dictionaries/:dictionaryName/search", controller.SearchDictionary)
	e.GET("/dictionaries/:dictionaryName/suggestions", controller.SuggestDictionary)

	addr := fmt.Sprintf("%s:%d", host, port)
	e.Logger.Fatal(e.Start(addr))
}
