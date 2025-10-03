package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
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

	fmt.Printf("Starting server on %s:%d\n", address, port)
	_, err = config.InitDB(dataDir, &conf)
	if err != nil {
		slog.Error("error while initializing DB", "err", err)
	}
}
