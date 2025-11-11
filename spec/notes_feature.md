## Feature: Collector's notes on scripture excerpts

`Excerpt` struct already has a notes field but it is unused.

I want to make it easy to store notes for each excerpt, by putting them in a flat markdown file formatted like this

```
## 1.4.5

My Notes Paragaraph 1

My Notes Paragraph 2
```

We don't need to index it right now. We can just store it.

Each `ScriptureDefinition` can _Optionally_ point to a "notes_file", which will be parsed using goldmark (already added as dependency). It should be parsed before loading the scripture into index in `bleve_initial_load.go`. Then when loading the actual excerpt, check if it has any excerpt in the map and attach it accordingly in `Notes` field after converting it into simple HTML.

# UI implementation
UI in `excerpts.html` can consider this notes field as safe and render it inside another card (after all auxiliaries) if it exists. 
