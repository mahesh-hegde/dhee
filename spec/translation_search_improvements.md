## Improvements to translation search

Implement the following enhancements in both bleve and SQLite3 based excerpt stores

1. The main translation (indicated by `TranslationAuxiliary`) should be a dedicated FTS / field, which is accent-folded and porter-stemmed.
    1. For SQLite, its not possible to use different tokenization for different columns. Therefore we will separate out the translations as a different FTS table, and update the queries accordingly. (We also don't need to ingest other auxiliaries - which will save some space in the existing FTS table).
    2. Make sure this is populated during `Add` in both bleve & sqlite3 excerpt stores. For bleve, you need to add a mapping in `initial_load.go`.
2. Search results should be highlighted
    1. When ingesting, HTML-escape all the text and auxiliary fields which are searched, before converting them to JSON.
    2. When searching, use fts highlight function with <strong>, </strong>
    3. When deserializing the JSON payload, override the queried field with its highlighted form.
    4. When rendering the template, use the raw/unescaped form.
    5. In the excerpts.templ, define `<strong>` to be highlighted in yellow, typical for search results.
