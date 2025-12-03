### Additional pages

Additional pages are the pages which are not generated from verse data, but created from *.md files in data/pages. They will be very short and can be held in memory.

Define a Page struct:
```
Title string
Content string
```

which should be initialized by calling a markdown parser at startup. Create a map where key is the filename without `*.md`.

It should be rendered through TemplateRenderer and layout.html for uniformity. Create new .templ file which has the following format.

```
[ToC]  [raw content div] [list of other pages]
```
