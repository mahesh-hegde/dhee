## Bleve to SQLite3 migration

We are using `bleve` as the backing store for our data but it has many disadvantages for us.

* It's not mature enough. There are times "How to do X in bleve" is not clear from search results.

* It doesn't let us store binary data. Some of our data is just dumb data (does not need indexing). It is beneficial to encode it as `gob` rather than `json` in the interest of space and performance.

* It is slow for SQL-type of queries.

We want to migrate our DictionaryStore to SQLite3 first.

### Steps (tentative - add more steps if required):

#### Step 1: Build Proper Abstractions for Store interfaces

* Update Load methods in bleve_initial_load.go to use the DocStore instead of bleve index directly.

* Create Init() method on BleveDictionaryStore and BleveExcerptStore, which create necessary schemas/mappings.

* Any other steps to make initial load logic independent of the docstore implementation, so that it can be reused.

#### Step 2: SQLite3 scaffolding and data load
* Create SQLite3 based DictionaryStore and ExcerptStore classes with stub methods.
* Replace bleve dependencies which are being injected by SQLite3
* Make it possible to switch between mattn sqlite3 (CGo based) and ModernC sqlite3 using a build tag in one file. (Since both are database/sql compatible)
* Create Init() methods for SQLite3
  * Create schemas in SQLite3 `dhee_excerpts` and `dhee_dictionary_entries`
  * Primary key will be string (since we are using `<scriptureId>:<index>` or `<dictId>:SLP1Word` format in bleve and don't want to disrupt it).
  * Create FTS5 virtual table to store full text indexes. But use simple columns in the main table for keyword fields (fields which are non-analyzed in bleve).
  * For multi-valued columns (ExcerptInDB.authors, DictionaryEntryInDB.variants etc..), create an FTS column which stores all values separated by `, `.
  * The main excerpt/dictionary entry payload should be serialized as `gob` to save space.
  * Create a spellfix1 table for fuzzy search on dictionary and populate it using dictionary words.

* Create `New____StoreWithSQLite()` which take a higher level object (like a connection pool) which can be shared across concurrent requests.
  * The db path should always be '{dataDir}/dhee.db'.

* Make sure `init` runs successfully and loads data into an SQLite DB.

* Add command line switches for `server`, `index` and `stats` commands in main, which can switch between SQLite3 and Bleve implementations. Define the default in global variable.

#### Step 3 Implement store methods in `DictionaryStore` and `ExcerptStore`.
  * Mirror the existing Bleve implementations in new file (same package).
    * (We will keep bleve implementation for now, do not delete it yet).
    * Follow the sorting behavior properly.
    * Keep the queries optimal.
    * Fuzzy search on excerpts can be skipped with an error message "unsupported".
