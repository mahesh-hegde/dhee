# Dhee - A platform for linguistic analysis of Vedic Sanskrit texts

Dhee is a platform for studying and analyzing old vedic sanskrits. Currently Rigveda is supported. The long term goal is to support any vedic sanskrit text with a well defined chapter/verse hierarchy and English translations.

Design goals
- Simple UI (technology-wise; no SPA, no NPM, no bloat).
- Performant backend (written in Go).
- Extensible enough to support texts other than the Rigveda.

## Implementation roadmap

### Short term

- [X] View one / many verses directly along with translations
- [ ] Search (regexp and / or text based - half done, there are some bugs due to accent marks in the dataset).
- [ ] Show Monier-Williams dictionary hints along with Padapatha text.
- [ ] Hierarchical navigation (i.e show the mandala/sukta/rik hierarchy).

### Long term
- [ ] Semantic search and similarity scores (hybrid embeddings + BM25, embeddings generated at build time)
- [ ] Graphing and visualization wizard using `d3js` / `uplot`, for analyzing word frequency and grammatical forms.

### Very long term
- [ ] Find and include data for Yajurveda and Atharvaveda samhitas.
- [ ] port INRIA's nominal declension and verb conjugation generator to Go and use it to analyze arbitrary word forms.


## How to run?
```bash
# create a bleve search index of all data
go run ./cmd/dhee index --data-dir ./data
# run server
go run ./cmd/dhee server --data-dir ./data
```

## Acknowledgements

Much of the data present now is taken from from [VedaWeb data](https://github.com/VedaWebProject/vedaweb-data/tree/main/rigveda) and [Monier Williams dictionary](https://www.sanskrit-lexicon.uni-koeln.de/) by Cologne university.

Some transliteration data was taken / adapted from [indic-translation](https://github.com/indic-transliteration/common_maps).

Favicon from Anil Sharma on Pixabay: https://pixabay.com/photos/eagle-bird-golden-eagle-bird-flying-6979972/

As of present state of this project (WIP), Cologne VedaWeb's `tekst` may be indeed a better resource for any serious analysis.
