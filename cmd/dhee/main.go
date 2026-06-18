package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"

	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/docstore"
	"github.com/mahesh-hegde/dhee/app/excerpts"
	"github.com/mahesh-hegde/dhee/app/server"
	"github.com/mahesh-hegde/dhee/app/transliteration"
	"github.com/spf13/pflag"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	logLevel := os.Getenv("LOG_LEVEL")
	slogLevel := slog.LevelInfo
	switch logLevel {
	case "DEBUG", "debug":
		slogLevel = slog.LevelDebug
	case "WARN", "warn":
		slogLevel = slog.LevelWarn
	}

	slog.SetLogLoggerLevel(slogLevel)

	command := os.Args[1]

	switch command {
	case "preprocess":
		runPreprocess()
	case "server":
		runServer()
	case "lambda":
		runLambda()
	case "index":
		runIndex()
	case "stats":
		runStats()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: dhee <command> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  preprocess    Convert input data to output format")
	fmt.Fprintln(os.Stderr, "  server        Start the dhee server")
	fmt.Fprintln(os.Stderr, "  lambda        Start as AWS Lambda function")
	fmt.Fprintln(os.Stderr, "  index         Build the search index in advance")
	fmt.Fprintln(os.Stderr, "  stats         Show index statistics")
}

func readConfig(dataDir string) *config.DheeConfig {
	confPath := path.Join(dataDir, "config.json")
	confFile, err := os.Open(confPath)
	if err != nil {
		slog.Error("error while opening config.json", "err", err)
		os.Exit(1)
	}
	defer confFile.Close()

	var conf config.DheeConfig
	confDec := json.NewDecoder(confFile)
	if err := confDec.Decode(&conf); err != nil {
		slog.Error("error while reading config.json", "err", err)
		os.Exit(1)
	}
	return &conf
}

func runPreprocess() {
	flags := pflag.NewFlagSet("preprocess", pflag.ExitOnError)
	var input, output, embeddingsFile string
	flags.StringVarP(&input, "input", "i", "", "Input directory (required)")
	flags.StringVarP(&output, "output", "o", "", "Output directory (required)")
	flags.StringVar(&embeddingsFile, "embeddings-file", "", "Path to embeddings JSONL file (optional)")

	flags.Parse(os.Args[2:])

	if input == "" || output == "" {
		fmt.Fprintln(os.Stderr, "Error: --input and --output are required")
		os.Exit(1)
	}

	mwInput := path.Join(input, "mw.xml")
	mwOutput := path.Join(output, "mw.jsonl")

	if err := dictionary.ConvertMonierWilliamsDictionary(mwInput, mwOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := excerpts.PreprocessRvDataset(path.Join(input, "tei"), output, embeddingsFile); err != nil {
		slog.Error("error when preprocessing rigveda dataset", "error", err)
		os.Exit(1)
	}
}

func runServer() {
	flags := pflag.NewFlagSet("server", pflag.ExitOnError)
	var dataDir string
	var cpuProfile, memProfile string
	var serverConf config.ServerRuntimeConfig

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(jsonHandler))

	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.StringVar(&cpuProfile, "cpu-profile", "", "write cpu profile to file")
	flags.StringVar(&memProfile, "mem-profile", "", "write memory profile to file")

	flags.StringVarP(&serverConf.Addr, "address", "a", "localhost", "Server address to bind")
	flags.IntVarP(&serverConf.Port, "port", "p", 8080, "Server port to bind")
	flags.StringVar(&serverConf.CertDir, "cert-dir", "", "directory to read/write TLS certs for ACME")
	flags.BoolVar(&serverConf.AcmeEnabled, "acme", false, "use ACME to renew TLS certificates")
	flags.BoolVar(&serverConf.BehindLoadBalancer, "behind-load-balancer", false, "Certain behaviors when behind a load balancer (e.g., trusting X-Forwarded-For header)")
	flags.IntVar(&serverConf.GzipLevel, "gzip-level", 1, "Gzip compression level (1-9), or 0 to disable gzip")
	flags.IntVar(&serverConf.RateLimit, "rate-limit", 0, "Number of requests per second for rate limiting")
	flags.IntVar(&serverConf.GlobalRateLimit, "global-rate-limit", 0, "Global request rate limit per second")

	flags.Parse(os.Args[2:])

	var cpuProfFile *os.File
	if cpuProfile != "" {
		var err error
		cpuProfFile, err = os.Create(cpuProfile)
		if err != nil {
			slog.Error("could not create CPU profile", "err", err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
			slog.Error("could not start CPU profile", "err", err)
			os.Exit(1)
		}
	}

	if cpuProfile != "" || memProfile != "" {
		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt)
			<-sigs
			slog.Info("interrupt received, writing profiles and shutting down")

			if memProfile != "" {
				f, err := os.Create(memProfile)
				if err != nil {
					slog.Error("could not create memory profile", "err", err)
				} else {
					runtime.GC()
					if err := pprof.WriteHeapProfile(f); err != nil {
						slog.Error("could not write memory profile", "err", err)
					}
					f.Close()
					slog.Info("memory profile written", "file", memProfile)
				}
			}

			if cpuProfile != "" {
				pprof.StopCPUProfile()
				cpuProfFile.Close()
				slog.Info("cpu profile written", "file", cpuProfile)
			}
			os.Exit(0)
		}()
	}

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}

	conf := readConfig(dataDir)
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
	server.StartServer(controller, conf, serverConf)
}

func runIndex() {
	flags := pflag.NewFlagSet("index", pflag.ExitOnError)
	var dataDir string
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}
	conf := readConfig(dataDir)

	slog.Info("starting indexing", "data-dir", dataDir)
	closer, err := docstore.InitDB("sqlite", dataDir, conf)
	if err != nil {
		slog.Error("error while initializing store", "err", err)
		os.Exit(1)
	}
	if closer != nil {
		if err := closer.Close(); err != nil {
			slog.Error("error closing store", "err", err)
		}
	}
	slog.Info("finished indexing")
}

func runStats() {
	flags := pflag.NewFlagSet("stats", pflag.ExitOnError)
	var dataDir string
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}

	db, err := docstore.NewSQLiteDB(dataDir, true)
	if err != nil {
		slog.Error("error while initializing SQLite DB", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	var count int

	// Dictionary entries
	err = db.QueryRow("SELECT COUNT(*) FROM dhee_dictionary_entries").Scan(&count)
	if err != nil {
		slog.Error("error getting dictionary entry count", "err", err)
	} else {
		fmt.Printf("'dictionary_entry' count: %d\n", count)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM dhee_excerpts").Scan(&count)
	if err != nil {
		slog.Error("error getting excerpt count", "err", err)
	} else {
		fmt.Printf("'scripture' count: %d\n", count)
	}
}
