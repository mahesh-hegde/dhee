package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

func StartServer(controller *DheeController, conf *config.DheeConfig, host string, port int) {
	e := echo.New()
	e.Renderer = NewTemplateRenderer(conf)
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		msg := http.StatusText(code)

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			if he.Message != nil {
				msg = fmt.Sprintf("%v", he.Message)
			}
		}

		if he, ok := err.(*common.UserVisibleError); ok {
			code = he.HttpCode
			msg = he.Error()
		}

		c.Logger().Error(err)

		if !c.Response().Committed {
			if renderErr := c.Render(code, "error", msg); renderErr != nil {
				c.Logger().Error(renderErr)
			}
		}
	}
	e.HideBanner = true

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	if conf.TimeoutSeconds != 0 {
		e.Use(middleware.ContextTimeout(time.Duration(conf.TimeoutSeconds) * time.Second))
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		LogLatency:  conf.LogLatency,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Int64("latency_ms", v.Latency.Milliseconds()),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
					slog.Int64("latency_ms", v.Latency.Milliseconds()),
				)
			}
			return nil
		},
	}))

	staticDir, err := fs.Sub(staticFs, "static")
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(staticDir)))))

	e.GET("/favicon.ico", func(c echo.Context) error {
		file, err := templateFs.ReadFile("templ_template/favicon.ico")
		if err != nil {
			// Let's not expose internal server errors, a simple 404 is sufficient
			return echo.ErrNotFound
		}
		return c.Blob(http.StatusOK, "image/x-icon", file)
	})

	e.GET("/", controller.GetHome)
	e.GET("/scriptures/:scriptureName/excerpts/:path", controller.GetExcerpts).Name = "excerpts"
	e.GET("/scriptures/:scriptureName/excerpts", controller.GetExcerpts)
	e.GET("/scriptures/:scriptureName/hierarchy", controller.GetHierarchy)
	e.GET("/scriptures/:scriptureName/hierarchy/:path", controller.GetHierarchy).Name = "hierarchy"
	e.GET("/scripture-search", controller.SearchScripture)
	e.GET("/dictionaries/:dictionaryName/words/:word", controller.GetDictionaryWord)
	e.GET("/dictionaries/:dictionaryName/search", controller.SearchDictionary)
	e.GET("/dictionaries/:dictionaryName/suggestions", controller.SuggestDictionary)

	addr := fmt.Sprintf("%s:%d", host, port)
	e.Logger.Fatal(e.Start(addr))
}
