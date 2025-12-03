## Search UI Improvements

We offer a search form in home `home.templ`. It can be used as dictionary search or excerpt search (leads to html result of `dictionary_search.html` or `scripture_search.templ` respectively).

The search as it stands today is pretty primitive. We need to improve it.

1. Save and set the search string encoding preference globally (per scripture, use a JSON map).
  - use localstorage to switch encoding preference when new value is selected.
  - update all encoding preference dropdowns once it changes. So that the change in encoding would be global.
  - the encoding dropdown (SLP1, IAST ...) should use INFO background to denote.
  - when search type is "translation", mute the encoding box.

2. Show search bar on top of dictionary search page, with current entry and search settings.
  - To enable this, we need to extract the dictionary search widget from home page into a separate reusable `templ` file.
  - We can pass additional values to the template from the controller if current data are not enough.

3. Similarly, show search bar on top of scripture search screen (`scripture_search.templ`), so that subsequent searches can be performed without navigating back to the home page.

4. Similarly, show the same search bar on top of the `excerpts.html` and `dictionary_word.templ` pages, to avoid needing to navigate to home page. The content should be empty,

5. Now since these reusable components may need their own JS, create `preInitFns` array and check if before calling `initFn()` in `layout.html`.

Note that all of these should have valid and uniform defaults before preInitFns get executed. i.e all these preInitFns may not run if user disabled javascript.
