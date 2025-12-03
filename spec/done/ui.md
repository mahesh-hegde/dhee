## UI Design draft

Use the following design draft to complete the templates in app/server/template folder.

Ask if any extra methods in services are needed to obtain the information for rendering.

That way I can review your design of those methods and suggest changes.

General instructions
* Use bootstrap 5
* Responsive design - readable on phone
* [v <description>] in the following designs represents a dropdown.
  * style the form elements with bootstrap as necessary.

Finally, any of the GET screens where no result is found should display an error (no results!) instead of blank.

## Home page

----------------------

[Logo]           Dhee
  A platform for linguistic analysis of
  vedic sanskrit texts

----------------------

Scriptures

[accordion] scripture 1

H3 Jump to excerpt
[Enter excerpt number separated by dots] [Go]

H3 Search
[ search text ] [v transliteration] [v exact/fuzzy/regex/prefix] [Go]

----------------------

Dictionaries

[accordion] Dictionary 1
[ search text ] [v transliteration] [v exact/fuzzy/regex/prefix] [Go]

------

About

[static text pasted from dedicated file]


## Scripture search results

```
GET /scripture-search/?params?query=<query>&tl=slp1&mode=exact
```

[nav bar same as home page]

(H2) results (table with text wrap for long results)

Path/readableIndex | Roman text | Translation (auxiliaries.griffith) | Addressee

## Scripture GET verse
/scriptures/{scriptureName}/excerpts/path
eg
GET /scriptures/rigveda/excerpts/4.1.1

----------------------------------------------------
sub-title
index: Devanagari source text line1
       Devanagari source text line2
       (lines are split and stored as array)
-------------
roman text (same way)
============
Auxiliary name:
  auxiliary content (same way)
============

or even multiple consecutive excerpts, in this case, each text version and auxiliary should render all verses together.
```
GET /scriptures/rigveda/excerpts/4.1.1-4
```

In this case, each subsection should have all verses together. eg: roman text should have 4.1.1 - 4.1.4 without other formats interspersing. The purpose is to take screenshots easily.

## Dictionary search results

```
GET /dictionaries/{dictionaryName}/search?q={query}&tl=slp1&mode=prefix
```

Word (IAST) | Word (devanagari) | Overview

## Dictionary GET word

There can be multiple hits for same word (because same word can have multiple meanings)

GET /dictionaries/{dictionaryName}/words/{slp1Word}

show all of them one by one, seperated by line break.

iast_word: body.plain
