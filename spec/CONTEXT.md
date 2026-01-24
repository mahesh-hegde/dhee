## What's this application
`dhee` is a website for studying and analyzing old indic texts, specifically Rigveda Samhita.

It's written in Go and we plan to support advanced features like search and visualizations. Currently we use `bleve` as a text search backend.

## Modules
* scripture reader
* dictionary

These 2 modules are kept separate. Scripture module can depend on dictionary module to retrieve meanings of the words.

The scripture reader is designed to support any sanskrit scripture with arbitrarily defined hierarchy (which can be configured as a scripture definition in `{data_dir}/config.json` and read from main). These definitions are read at startup and stored in `config.DheeConfig` struct.

## Data sources
* rigveda from vedaweb dataset
* monier-williams sanskrit-english dictionary

These are preprocessed into a standard intermediate format using `dhee preprocess` command. JSONL is chosen because of its streaming properties.

Then `dhee index` command is used to create a SQLite3 index which can be used by server to efficiently search and serve contents.

## How to verify your changes?

Depending on which part(s) of the app you are modifying, you can run the following commands to confirm.

- Preprocessing: `go run ./cmd/dhee preprocess --input ./data --output ./data`
  - This will convert the desperate data from various sources into JSONL files.

- Indexing data: `rm -rf data/dhee.db; go run ./cmd/dhee index --data-dir ./data`
  - This will index the JSONL data into a SQLite3 database in --data-dir.
  - removing the existing db is required to force refresh.

- Server `go run ./cmd/dhee server --data-dir ./data`
  - This will start serving on port 8080

## Misc Features
* Automatically linking Monier-williams dictionary entries with Padapatha, as popups for easy reading
* Ability to select part of roman text and search it in dictionary without switching pages, implemented in 
