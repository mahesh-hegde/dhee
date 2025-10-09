package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/docstore"
	"github.com/mahesh-hegde/dhee/app/scripture"
	"github.com/mahesh-hegde/dhee/app/server"
	"github.com/spf13/pflag"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

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

	if err := scripture.PreprocessRvDataset(path.Join(input, "tei"), output); err != nil {
		slog.Error("error when preprocessing rigveda dataset", "error", err)
		os.Exit(1)
	}
}

func runServer() {
	flags := pflag.NewFlagSet("server", pflag.ExitOnError)
	var address, dataDir string
	var port int
	flags.StringVarP(&address, "address", "a", "localhost", "Server address to bind")
	flags.IntVarP(&port, "port", "p", 8080, "Server port to bind")
	flags.StringVarP(&dataDir, "data-dir", "d", "",
		"data directory to read config.json and data JSONL files")

	flags.Parse(os.Args[2:])

	if dataDir == "" {
		slog.Error("--data-dir not provided, stopping")
		os.Exit(1)
	}
	conf := readConfig(dataDir)

	fmt.Printf("Starting server on %s:%d\n", address, port)
	index, err := docstore.InitDB(dataDir, conf)
	if err != nil {
		slog.Error("error while initializing DB", "err", err)
	}

	controller := server.NewDheeController(index, conf)
	server.StartServer(controller, conf, address, port)
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
	idx, err := docstore.InitDB(dataDir, conf)
	if err != nil {
		slog.Error("error while initializing DB", "err", err)
	}
	idx.Close()
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
}
