## Client side transliteration to IAST using JavaScript

We need a clone of the logic in `app/transliteration/transliteration.go` on client side as a javascript library. It will be used to

* provide instant feedback (as a small popup box when typing in SLP1/HK)
* let the user edit the query in selection-search before searching. This way user can edit the query in SLP1 / HK but preview it in IAST without a network call

## Steps
1. Create a file handler to serve static files embedded from app/server/static
2. Create a javascript file app/server/static/common.js with the transliteration functionality, providing the same interface as go `Transliterator` struct and mirroring its logic.
  * Leave placeholder for me to copy the slp1_mappings.json into this monolithic JS file.
3. Replicate the tests in `app/transliteration/transliteration_test.go` in JS file, with a simple assert function and simple test runner in < 20 lines, which can be run as `node app/server/static/transliteration_test.js` and should exit with zero or non-zero status accordingly.

3. Update the search boxes in `dictionary_search_widget.templ` and `scripture_search_widget.templ` to transliterate and display the query as a search suggestion on-the-fly. The suggestion text should also be selectable.
4. Update the selection search logic (check `selection_search.templ`) to update the simple search hint (the one prefixed with search emoji), into a more intricate card which appears in the top right of the selection. This way the user can edit eg: the inflected forms before searching. The select should honour the user preferences for transliteration type.

```
[ INPUT BOX with selected text ] [ SLP1 / HK / IAST / devanagari select in small size form element]

[[search-emoji] suggestion for IAST transliteration of the text in input box]
```
