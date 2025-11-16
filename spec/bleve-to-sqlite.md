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
  * Create FTS5 virtual table to store full text indexes. But use simple columns for fields which are non-analyzed in bleve.
  * Create a spellfix table for fuzzy search and populate it using FTS5 vocabulary after loading.

* After this step, make sure `init` runs successfully and loads data into an SQLite DB.

#### Step 3 Implement store methods in `DictionaryStore` and `ExcerptStore`.
  * Mirror the existing Bleve implementations in new file (same package).
    * (We will keep bleve implementation for now, do not delete it yet).
    * Follow the sorting behavior properly.
    * Do not make queries which result in a full table scan except in case of advanced / fuzzy search.
