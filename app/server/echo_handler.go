package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

// BuildEchoHandler creates and configures an Echo instance with the core routing and minimal middleware.
// This handler can be used by both the traditional HTTP server and Lambda runtime.
// It applies only the essential middleware: recovery, request ID, request logging, gzip (if enabled), and timeout.
// TLS, rate limiting, and hostname redirect middleware are NOT applied here (added by caller if needed).
func BuildEchoHandler(controller *DheeController, dheeConf *config.DheeConfig, serverConf config.ServerRuntimeConfig) (*echo.Echo, error) {
	e := echo.New()

	// Custom error handler
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

	// Core middleware: recovery, request ID, logging
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	e.Pre(middleware.RemoveTrailingSlash())

	if serverConf.BehindLoadBalancer {
		e.IPExtractor = echo.ExtractIPFromRealIPHeader()
	}

	// Optional: Gzip compression
	if serverConf.GzipLevel != 0 {
		e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: serverConf.GzipLevel, MinLength: 512}))
	}

	// Optional: Context timeout
	if dheeConf.TimeoutSeconds != 0 {
		e.Use(middleware.ContextTimeout(time.Duration(dheeConf.TimeoutSeconds) * time.Second))
	}

	// Setup renderer and static files
	staticDir, err := fs.Sub(staticFs, "static")
	if err != nil {
		return nil, err
	}

	staticServerHashFs, err := NewHashFS(staticDir)
	if err != nil {
		return nil, err
	}

	e.Renderer = NewTemplateRenderer(dheeConf, staticServerHashFs)

	// Register routes
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", staticServerHashFs)))

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

	return e, nil
}

// GetHTTPHandler returns the HTTP handler from an Echo instance.
// This handler can be used to serve HTTP requests directly (e.g., in Lambda).
func GetHTTPHandler(e *echo.Echo) http.Handler {
	return e
}
