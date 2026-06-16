package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/docstore"
	"github.com/mahesh-hegde/dhee/app/excerpts"
	"github.com/mahesh-hegde/dhee/app/server"
	"github.com/mahesh-hegde/dhee/app/transliteration"
	"github.com/spf13/pflag"
)

// Response size limit: 2MB
const MAX_RESPONSE_SIZE = 2 * 1024 * 1024

// LambdaHandlerState holds the initialized services for Lambda
type LambdaHandlerState struct {
	httpHandler interface{} // http.Handler
	initialized bool
}

func runLambda() {
	flags := pflag.NewFlagSet("lambda", pflag.ExitOnError)
	var dataDir string
	var serverConf config.ServerRuntimeConfig

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(jsonHandler))

	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.IntVar(&serverConf.GzipLevel, "gzip-level", 1, "Gzip compression level (1-9), or 0 to disable gzip")
	flags.BoolVar(&serverConf.BehindLoadBalancer, "behind-load-balancer", false, "Certain behaviors when behind a load balancer (e.g., trusting X-Forwarded-For header)")

	// Note: Lambda doesn't support custom timeout via flag, but we keep it for consistency
	// (would need to be set via Lambda environment variable or handler timeout)
	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}

	conf := readConfig(dataDir)

	// Initialize services once (will be reused across Lambda invocations)
	var dictStore dictionary.DictStore
	var excerptStore excerpts.ExcerptStore
	var err error

	db, err := docstore.NewSQLiteDB(dataDir, true)
	if err != nil {
		slog.Error("error while initializing SQLite DB", "err", err)
		os.Exit(1)
	}
	dictStore = dictionary.NewSQLiteDictStore(db, conf)
	excerptStore = excerpts.NewSQLiteExcerptStore(db, conf)

	transliterator, err := transliteration.NewTransliterator(transliteration.TlOptions{})
	if err != nil {
		slog.Error("error while initializing transliterator", "err", err)
		os.Exit(1)
	}

	controller := server.NewDheeController(dictStore, excerptStore, conf, &serverConf, transliterator)

	// Build the echo handler (without starting the server)
	echoHandler, err := server.BuildEchoHandler(controller, conf, serverConf)
	if err != nil {
		slog.Error("error while building echo handler", "err", err)
		os.Exit(1)
	}

	httpHandler := server.GetHTTPHandler(echoHandler)

	// Create the Lambda handler wrapper
	handler := func(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		startTime := time.Now()
		slog.DebugContext(ctx, "Lambda event received", "method", event.RequestContext.HTTP.Method, "path", event.RawPath)

		// Convert Lambda event to HTTP request
		req, err := server.LambdaEventToHTTPRequest(ctx, event)
		if err != nil {
			slog.ErrorContext(ctx, "error converting Lambda event to HTTP request", "err", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 400,
				Body:       "Bad Request",
				Headers:    map[string]string{"Cache-Control": "max-age=10800"},
			}, nil
		}

		// Create a buffered response writer with 2MB limit
		rw := server.NewBufferedResponseWriterWithLimit(MAX_RESPONSE_SIZE)

		// Invoke the HTTP handler
		httpHandler.ServeHTTP(rw, req)

		// Check if response was truncated due to size limit
		if rw.IsTruncated() {
			slog.WarnContext(ctx, "response truncated due to size limit", "limit_bytes", MAX_RESPONSE_SIZE, "written_bytes", rw.BytesWritten())

			// Replace with error response
			rw.Body.Reset()
			rw.Headers = make(http.Header)
			rw.StatusCode = 413 // StatusPayloadTooLarge

			errorHTML := `<!DOCTYPE html>
<html>
<head>
	<title>Response Too Large</title>
	<style>body { font-family: sans-serif; margin: 2em; }</style>
</head>
<body>
	<h1>Error 413: Payload Too Large</h1>
	<p>The requested page generated a response larger than the 2MB limit.</p>
	<p><a href="/">Return to home</a></p>
</body>
</html>`
			rw.Headers.Set("content-type", "text/html; charset=utf-8")
			rw.Body.WriteString(errorHTML)
		}

		// Add cache-control header for 2xx and 4xx responses
		if (rw.StatusCode >= 200 && rw.StatusCode < 300) || (rw.StatusCode >= 400 && rw.StatusCode < 500) {
			rw.Headers.Set("Cache-Control", "max-age=10800")
		}

		// Build URL with query string
		url := event.RawPath
		if event.RawQueryString != "" {
			url = url + "?" + event.RawQueryString
		}

		// Calculate processing time
		processingTime := time.Since(startTime)

		// Log request details
		slog.InfoContext(ctx, "Lambda request completed",
			"remote_ip", event.RequestContext.HTTP.SourceIP,
			"url", url,
			"status", rw.StatusCode,
			"response_size_bytes", rw.BytesWritten(),
			"processing_time_ms", processingTime.Milliseconds(),
		)

		// Convert response to Lambda format
		response := server.HTTPResponseToLambdaResponse(rw)
		slog.DebugContext(ctx, "Lambda response prepared", "status", response.StatusCode, "size_bytes", rw.BytesWritten())

		return response, nil
	}

	// Start the Lambda runtime
	slog.Info("starting Lambda runtime")
	lambda.Start(handler)
}
