Context is attached in CONTEXT.md. You can refer to schema.go. `ew` is `ExcerptWithWords`.

I want the excerpts.html edited to show hover cards on grammatical analysis table entries according to the below specification.

Provide edited version of excerpts.html 

## Implementing hover cards in "Grammatical Analysis" table entries.

We show hover cards on the words of "pada" auxiliary. But the hover on `surface` and `lemma` columns of the grammatical analysis table is too simple - its just a tooltip, with multiple entries text concatenated with ';'.

We can utilize same method to build on-hover cards for the table cells. In fact we are already looking up the dictionary information.

However, its also going to be slightly different from the "pada" hover cards. So I would not prematurely abstract the widget.

Here's the layout: notice no badges since badges are already in the table.

```
entry1
------
entry2
------
...
------
entryN
------
<a href=>[search emoji] {{ word }}</a>
```

the search should use IAST as the transliteration (TL), eg: "/dictionaries/monier-williams/search?q={{word}}&tl=iast&mode=fuzzy".

`word` of course refers to the word or lemma being highlighted, which we are already looking up.