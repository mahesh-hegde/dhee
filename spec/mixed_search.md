## Mixed-fuzzy-search using bleve and SQLite3

Go bindings to SQLite do not support spellfix1, which means we still need bleve to implement fuzzy search over dictionary words.

For this we will use a small bleve index at `{dataDir}/wordstore.bleve`.
* We define a simple struct with SLP1 word as ID
```
type BleveWord {
    slp1Word string `json:"slp1_word"`
    iastWord string `json:"iast_word"`
    slp1Variants []string `json:"slp1_variants"`
    iastVariants []string `json:"iast_variants"`
}
```

we use slp1Word as ID, but it doesn't really matter for fuzzy search in bleve.

* When constructing SQLite3 based `SQLiteDictionaryStore` and `SQLiteExcerptStore`, we
  * open an index for this word store
  * define keyword mappings for members of the above struct
  * as we put dictionary entries in SQLite3, we put the words and variants.

* When we need to do a fuzzy query in SQLite3 dictionary store.
  * Query max 100 fuzzy matches (either iast_word matches - boost by factor 2, or a variant matches) in the word store.
  * For this use fuzziness = 2. (define constant)
  * Now we have a list of 100 words which fuzzy-matched the query word. format them as IDs and query in SQLite3.
