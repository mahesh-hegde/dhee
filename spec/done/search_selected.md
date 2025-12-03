## Feature: search selected roman text in the dictionary
We provide search tooltips in the hover cards for
  1. cells in grammatical analysis table.
  2. recognized words in Pada auxiliary text

However this is not enough, we should provide more tools.

While that's nice, there are cases when I want to select arbitrary romanized (IAST) text and search that in the dictionary without switching the UI context.

I want to implement this feature as follows.

* In the roman text section of `excerpts.html`, any selection (which does not contain more than 2 whitespace characters) will trigger a hover popup with a search badge on top right corner.
  * The search badge is formatted as bootstrap btn, with text `<search emoji> <selected text>`, but unlike in other places, it wont link anywhere.
* When this badge is clicked, a request will be made to `/dictionaries/monier-williams/search?q={{ $g.Surface }}&tl=iast&mode=prefix&preview=true`, since preview is specified, the server will return a div fragment rather than full HTML page.
* This page should be displayed in a window which is no wider than 40% of the viewport, no taller than 50% of the viewport, and scrollable.

The preview is already implemented.
