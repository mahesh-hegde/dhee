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

Then `dhee index` command is used to create a bleve index which can be used by server to efficiently search and serve contents.
