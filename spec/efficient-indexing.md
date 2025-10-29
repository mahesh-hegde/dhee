Your context is provided in `CONTEXT.md`. Go source files are attached. If any struct or other type definitions required for this task are missing from attached files, ask before proceeding (static typing for the win!).

other files you have for are:
* `scriptures/schema.go`
* `dictionary/schema.go`
* `bleve_initial_load.go`

## Task: Efficient indexing with bleve

Presently we store `DictionaryEntry` and `Excerpt` types directly in `bleve` document store. 

* This leads to every field being indexed wastefully. 

* Moreover, original structure being lost due to flattening action (just like lucene/elasticsearch) by the bleve library. This requires overly complex deserialization implementations.

* The fields which are dealt with as `[]string` in business logic (eg: since they can spawn multiple lines), still need to be analyzed together. But that's not happening because storage format == object format.

In order to elide this restriction, two new types are defined which represent the back-and-forth between bleve and application. those types are `DictionaryEntryInDB` and `ExcerptInDB`.

Each of them have one field which stores the whole field as JSON but does not index it. Rest of the fields are used for indexing and need to be computed when indexing. But we don't need to deserialize them. When deserializing, we can just pick the JSON-serialized map member.

You need to perform the following actions

* Define methods `prepareDictEntryForDb(e *DictionaryEntry) DictionaryEntryInDB` and `prepareExcerptForDb(e *excerpt) ExcerptInDB`.
  * When a field is an array type in source but string in DB, it should be joined with single space character.
  * Update other loading methods minimally to make sure these methods are called.

* Update the mappings in `bleve_initial_load.go` to reflect the new `_InDB` types. Follow the requirements expressed here and field comments.

* In the same file, provide 2 public helpers
    1. func docToDictEntry(fields map[string]interface{}) (DictionaryEntry, error)
    2. func docToExcerpt(fields map[string]any) (Excerpt, error)

    so that any query can make use of the shared logic to fetch the correct type.

All your changes should be in `bleve_initial_load.go`. Provide an updated version of that file.

Notes:
* Do not hand code folding logic, bleve supports an ascii folding variant and we have already defined an ASCII folding analyzer in bleve.
* Utilize the analyzers already defined.
