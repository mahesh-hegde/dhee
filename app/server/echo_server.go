package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mahesh-hegde/dhee/app/config"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/time/rate"
)

func StartServer(controller *DheeController, dheeConf *config.DheeConfig, serverConf config.ServerRuntimeConfig) {
	// Build the core echo handler with routing and essential middleware
	e, err := BuildEchoHandler(controller, dheeConf, serverConf)
	if err != nil {
		slog.Error("failed to build echo handler", "err", err)
		os.Exit(1)
	}

	// Add server-specific middleware and configuration (not used in Lambda)
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

	if serverConf.GlobalRateLimit > 0 {
		e.Use(controller.GlobalRateLimitMiddleware)
	}

	host := serverConf.Addr
	port := serverConf.Port
	certDir := serverConf.CertDir
	acme := serverConf.AcmeEnabled

	addr := fmt.Sprintf("%s:%d", host, port)

	if certDir != "" {
		e.Pre(middleware.HTTPSRedirect())
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
