## UI user preferences

We display several versions of a verse. Users would ideally like to customize whether to display of the various formats including roman text, source text and various auxiliaries.

We have to implement a row of check boxes in `excerpts.templ`, and display them on top right (50% right half, justified) which lets the users select the versions they want to display. Store this as a JSON array in local storage, and if Javascript is enabled, run a minimal javascript in <head> which just reads it and sets the CSS/attributes with minimal work (so that the display is consistent on initial load).

Rest of the JS can go to end of the body, i.e initFn.

If JS is disabled, all versions should be displayed, and checkboxes should have all elements clicked, but with disabled attribute.

The keys should be either fieldnames in struct (for non-auxiliary fields), or "aux-"+"fieldname" for auxiliary fields. Check them before displaying.

Finally, reordering the display content should be supported on client side. Which means when we render cards, we should give them IDs based on the the key we use in local storage, and let the client side logic reorder them on load if JavaScript is present.

When use drags and drops within the checkbox row (which can spawn multiple lines), if we dragged and dropped into first half of another checkbox item, then we should insert BEFORE that item. else we should insert AFTER that item. We should show a small + symbol during drag where the checkbox would be inserted.

You will only ever need to modify `excerpts.templ`. You can read `excerpt_service.go` and `app/excerpts/schema.go` to know the data which will be passed to this template.
