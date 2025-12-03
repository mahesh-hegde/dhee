## Displaying expansions of padapatha elements.

Since Padapatha is the version of Vedic texts with each word spelled out separately, we have a very good opportunity to ease the study of vedic texts, by mapping each pada word to a dictionary element. Currently there's a table at the end which displays the same information, but its cumbersome to read, and hover tooltip is limited (just text using "title").

Backend changes have been already done in schema.go to provide pada text for the verses. You have to implement the UI in excerpts.html so that:

* if the auxiliary name is "pada", conditionally use ExcerptWithWords.Padas element instead of the auxiliary itself (pada element itself contains the words, render them separated by pipe |).
* When doing so, if the pada.Found is true, provide user a way to understand the dictionary entries for both the exact word (pada.SurfaceMeanings) and the meanings of the root (pada.Lemma)
  * If javascript is enabled, display a simple preview pop-up with all elements of the dictionary entry, rendering them serially.
  * Regardless of javascript enablement, link to /dictionaries/monier-williams/{entry.Word} once for either surfacemeanings or lemmameanings (whichever exists, in that order of priority, or none).

When javascript is enabled, along with the dictionary entry and title in the meanings, also show the same series of badges for each pada entry as shown in the table at the end of the page. Use the .G element to get the grammatical properties `ExcerptGlossing` for that pada word.

Extract the existing badge series logic (which goes into last td) into a separate sub template to keep code duplication less. Also display the `kind` column as a label.

How would you store the arbitrary data for each pada in the HTML page without displaying it? My recommendation is to use hidden HTML elements plus minimum javascript DOM manipulation to show them on hover, so that server can precompute all the HTML. Keep it simple.
