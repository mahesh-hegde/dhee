# Dhee - A platform for linguistic analysis of Vedic Sanskrit texts

Dhee is a platform for studying and analyzing old vedic sanskrits. Currently Rigveda is supported. The long term goal is to support any Vedic sanskrit text with a well defined chapter/verse hierarchy and English translations.

Design goals
- Simple, efficient and useful UI (technology-wise: no SPA, no NPM, no need for server rendering, no bloat).

- Performant backend (written in Go and use `SQLite3` embedded database which resides on the same machine. aiming at <30ms response time for most pages).

- Extensible and general enough to support texts other than the Rigveda. (Please contact me if you know the datasets for Vedic Sanskrit texts other than Rigveda: by which I mean the samhitas or brahmanas).

## Implementation roadmap

### Short term

- [X] View one / many verses directly along with translations
- [X] Search (regexp and / or text based).
- [X] Hierarchical navigation (i.e show the mandala/sukta/rik hierarchy).
- [X] Show Monier-Williams dictionary hints along with Padapatha text.
- [X] Embedding and textual (TF-IDF) based recommendations of similar verses. (Currently using this model: `Snowflake/snowflake-arctic-embed-l-v2.0`)
- [ ] Integrate the [Multi-layer annotation of rigveda](https://ashutosh-modi.github.io/publications/papers/lrec18/Multi-layer%20Annotation%20of%20the%20Rigveda.pdf) to show shorter lexicon meanings before the dictionary entries.
- [ ] Integrate `anukramaNi` data on verse authors for rigveda.
- [ ] Use protocol buffer encoding in the SQLite database non-queriable blobs instead of JSON.

### Long term
- [X] Semantic similarity scores (embeddings and TF/IDF; embeddings generated at build time using a sentence transformer model and used to compute semantically similar verses.)
- [ ] Graphing and visualization wizard using `d3js` / `uplot`, for analyzing word frequency and grammatical forms across multiple scriptures using an advanced form input.
- [ ] Highlight and allow analysis of repeated refrains (N-gram where N >= 3)
- [ ] Advanced search using a custom query syntax (boolean operators, grouping and column filters)

### Very long term
- [ ] Find and include data for Yajurveda and Atharvaveda samhitas.
- [ ] Arbitrary embedding search
- [ ] port INRIA's inflected forms generator to Go and use it to analyze arbitrary word forms.
- [ ] Support auto detecting variations and verse references across texts.

## How to run?
```bash
# create a bleve search index of all data
go run ./cmd/dhee index --data-dir ./data
# run server
go run ./cmd/dhee server --data-dir ./data
```

## Regenerating embeddings
You will need python to generate embeddings

```bash
python3 ./script/cosine_similarity.py --input-file data/rv.jsonl --embedding-model Snowflake/snowflake-arctic-embed-l-v2.0 --output-file data/rv.emb.jsonl --auxiliaries griffith

go run ./cmd/dhee preprocess --input ./data --output ./data --embeddings-file ./data/rv.emb.jsonl
```

## Acknowledgements

Much of the data present now is taken from from [VedaWeb data](https://github.com/VedaWebProject/vedaweb-data/tree/main/rigveda) and [Monier Williams dictionary](https://www.sanskrit-lexicon.uni-koeln.de/) by Cologne university.

Some transliteration mappings was taken / adapted from [indic-transliteration](https://github.com/indic-transliteration/common_maps).

Favicon from Anil Sharma on Pixabay: https://pixabay.com/photos/eagle-bird-golden-eagle-bird-flying-6979972/

As of present state of this project (WIP), Cologne VedaWeb's `tekst` may be indeed a better resource for any serious analysis.
