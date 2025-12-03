## Feature: attributions at scripture and auxiliary level
* When rendering scripture in `excerpts.templ`, add attribution as pre-escaped HTML from scripture config at the end, center bottom of the screen with 50% font size.

* Add per-auxiliary attribution field.
* Add method to DheeConfig to get attribution map (auxiliary name -> attribution text).
* modify `renderer.go` if required to pass Config to `excerpts.templ`.
* When rendering auxiliary, check if there's a attribution specific to this auxiliary. If yes, add an attribution at bottom right of the auxiliary card in 45% font size.
