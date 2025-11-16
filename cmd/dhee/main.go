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

	"github.com/blevesearch/bleve/v2"
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
	var input, output string
	flags.StringVarP(&input, "input", "i", "", "Input directory (required)")
	flags.StringVarP(&output, "output", "o", "", "Output directory (required)")

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

	if err := excerpts.PreprocessRvDataset(path.Join(input, "tei"), output); err != nil {
		slog.Error("error when preprocessing rigveda dataset", "error", err)
		os.Exit(1)
	}
}

func runServer() {
	flags := pflag.NewFlagSet("server", pflag.ExitOnError)
	var address, dataDir, store string
	var port int
	var cpuProfile, memProfile string

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(jsonHandler))

	flags.StringVarP(&address, "address", "a", "localhost", "Server address to bind")
	flags.IntVarP(&port, "port", "p", 8080, "Server port to bind")
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.StringVar(&store, "store", "bleve", "storage backend to use (bleve or sqlite)")
	flags.StringVar(&cpuProfile, "cpu-profile", "", "write cpu profile to file")
	flags.StringVar(&memProfile, "mem-profile", "", "write memory profile to file")

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

	switch store {
	case "bleve":
		dbPath := path.Join(dataDir, "docstore.bleve")
		index, err := bleve.OpenUsing(dbPath, map[string]any{"read_only": true})
		if err != nil {
			slog.Error("error opening index, did you run the 'index' command first?", "err", err)
			os.Exit(1)
		}
		dictStore = dictionary.NewBleveDictStore(index, conf)
		excerptStore = excerpts.NewBleveExcerptStore(index, conf)
	case "sqlite":
		db, err := docstore.NewSQLiteDB(dataDir)
		if err != nil {
			slog.Error("error while initializing SQLite DB", "err", err)
			os.Exit(1)
		}
		dictStore = dictionary.NewSQLiteDictStore(db, conf)
		excerptStore = excerpts.NewSQLiteExcerptStore(db, conf)
	default:
		slog.Error("unknown store type", "store", store)
		os.Exit(1)
	}

	fmt.Printf("Starting server on %s:%d\n", address, port)

	transliterator, err := transliteration.NewTransliterator(transliteration.TlOptions{})
	if err != nil {
		slog.Error("error while initializing transliterator", "err", err)
		os.Exit(1)
	}

	controller := server.NewDheeController(dictStore, excerptStore, conf, transliterator)
	server.StartServer(controller, conf, address, port)
}

func runIndex() {
	flags := pflag.NewFlagSet("index", pflag.ExitOnError)
	var dataDir, store string
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.StringVar(&store, "store", "bleve", "storage backend to use (bleve or sqlite)")
	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}
	conf := readConfig(dataDir)

	slog.Info("starting indexing", "data-dir", dataDir, "store", store)
	closer, err := docstore.InitDB(store, dataDir, conf)
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
	var dataDir, store string
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")
	flags.StringVar(&store, "store", "bleve", "storage backend to use (bleve or sqlite)")
	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}

	if store == "bleve" {
		dbPath := path.Join(dataDir, "docstore.bleve")
		slog.Info("opening DB", "dbPath", dbPath)
		index, err := bleve.Open(dbPath)
		if err != nil {
			slog.Error("error opening index, did you run the 'index' command first?", "err", err)
			os.Exit(1)
		}
		defer index.Close()

		docTypes := map[string]string{"dictionary_entry": "dict_name", "scripture": "scripture"}
		for docType, field := range docTypes {
			// query := bleve.NewTermQuery(docType)
			// query.SetField("_type")
			query := bleve.NewRegexpQuery(".+")
			query.SetField(field)
			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"*"}
			searchRequest.Size = 1 // We only need the count
			searchResult, err := index.Search(searchRequest)
			if err != nil {
				slog.Error("error searching for doc type", "type", docType, "err", err)
				continue
			}
			fmt.Printf("'%s' count: %d\n", docType, searchResult.Total)
		}
	} else if store == "sqlite" {
		fmt.Println("stats for sqlite not implemented yet")
	} else {
		slog.Error("unknown store type", "store", store)
		os.Exit(1)
	}
}
