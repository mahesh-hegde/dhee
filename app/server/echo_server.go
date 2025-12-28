package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/time/rate"
)

func StartServer(controller *DheeController, dheeConf *config.DheeConfig, serverConf config.ServerRuntimeConfig) {
	e := echo.New()
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
	e.Pre(middleware.HTTPSRedirect())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			url := req.URL
			if req.Host != dheeConf.Hostnames[0] && serverConf.AcmeEnabled {
				// Redirect
				url.Host = dheeConf.Hostnames[0]
				slog.Info("redirect to canonical hostname", "original_hostname", req.Host)
				return c.Redirect(http.StatusPermanentRedirect, url.String())
			}
			return next(c)
		}
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	var identifierExtractor middleware.Extractor

	if serverConf.BehindLoadBalancer {
		identifierExtractor = func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		}
	} else {
		identifierExtractor = func(ctx echo.Context) (string, error) {
			id := ctx.Request().RemoteAddr
			return id, nil
		}
	}

	// configure rate limiting if enabled
	if serverConf.RateLimit > 0 {
		config := middleware.RateLimiterConfig{
			Skipper: middleware.DefaultSkipper,
			Store: middleware.NewRateLimiterMemoryStoreWithConfig(
				middleware.RateLimiterMemoryStoreConfig{
					Rate:      rate.Limit(serverConf.RateLimit),
					Burst:     3 * serverConf.RateLimit,
					ExpiresIn: 3 * time.Minute,
				},
			),
			IdentifierExtractor: identifierExtractor,
			ErrorHandler: func(context echo.Context, err error) error {
				return context.String(http.StatusForbidden, "Forbidden")
			},
			DenyHandler: func(context echo.Context, identifier string, err error) error {
				return context.String(http.StatusTooManyRequests, "Too Many Requests")
			},
		}

		e.Use(middleware.RateLimiterWithConfig(config))
	}

	if serverConf.GzipLevel != 0 {
		e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: serverConf.GzipLevel, MinLength: 512}))
	}

	if dheeConf.TimeoutSeconds != 0 {
		e.Use(middleware.ContextTimeout(time.Duration(dheeConf.TimeoutSeconds) * time.Second))
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		LogRemoteIP: true,
		LogLatency:  dheeConf.LogLatency,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Int64("latency_ms", v.Latency.Milliseconds()),
					slog.String("remote_ip", v.RemoteIP),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
					slog.String("remote_ip", v.RemoteIP),
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

	staticServerHashFs, err := NewHashFS(staticDir)
	if err != nil {
		e.Logger.Fatal(err)
	}

	e.Renderer = NewTemplateRenderer(dheeConf, staticServerHashFs)

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

	host := serverConf.Addr
	port := serverConf.Port
	certDir := serverConf.CertDir
	acme := serverConf.AcmeEnabled

	addr := fmt.Sprintf("%s:%d", host, port)

	if certDir != "" {
		if acme {
			slog.Info("using TLS with ACME", "dir", certDir)
			e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(dheeConf.Hostnames...)
			e.AutoTLSManager.Cache = autocert.DirCache(certDir)
			e.Logger.Fatal(e.StartAutoTLS(addr))
		} else {
			slog.Info("using TLS with certDir", "dir", certDir)
			e.Logger.Fatal(e.StartTLS(addr, path.Join(certDir, "fullchain.pem"), path.Join(certDir, "privkey.pem")))
		}
	} else {
		e.Logger.Fatal(e.Start(addr))
	}
}
