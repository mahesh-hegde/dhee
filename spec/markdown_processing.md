## Update markdown processing logic

* Add `markdown_converter.go` adjacent to `initial_load.go`.

* Create a MarkdownConverter struct with any state required. It should have a single `ConvertToHTML(text string, scripture config.ScriptureDefn)` method which does the following.
  * Convert the italicized words (without space) from HK into `IAST` encoding using a `Transliterator`.
  * Convert single-backquote formatted (eg: `word`) from HK into IAST, if they exist in dictionary, link them with single underline.
  * Convert relative links without a protocol, starting with @scripture_name#x.y, into links of the form `/scriptures/<scripture_name>/x.y` (for any length of dot-separated sequence x.y.z....)
  * To do the dictionary checks, you would need a DictionaryStore implementation, so it can be taken as a parameter.

    ```go
    func NewMarkdownConverter(d *dictionary.DictionaryStore, t *transliteration.Transliterator) *MarkdownConverter {
        // ....
    }
    ```
