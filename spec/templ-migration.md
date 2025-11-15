## Migrate from golang html/template to `templ`

We use templates in app/server/templates, it has 2 disadvantages:

* template rendering occupies almost 50% time in our profile.
* it is runtime reflection based and thus is not type-safe.

So we need to migrate to `templ` - which is a code generation based templating language for Go, providing better type safety and performance.

## Instructions

* Refer to the templ documentation above, catted from a file `docs_condensed.md`.

* Write templ files in app/server/templ_template/ for each existing HTML template file in app/server/template. They should be drop-in replacements for existing page templates / sub-templates.
  - You need to know the types which are input to each template. For that, refer to `controllers.go` which is attached, and the service methods called by it. Service method interface declaration is attached below, for more detailed definition of these structs you can see respective `schema.go` files.
  - templ supports functions and fine grained composition unlike html/template. Thus inside a tempalte you can refactor the code as functions.
  - But do not make organizational changes since we want drop-in replaceability of text templates by `templ` functions in existing code.

* Update the `TemplateRenderer` struct to use templ functions rather than the root html/template.

* Add a `go generate` directive in same file as template renderer.

## Style guidelines

* Do not define types in templ files unless they're purely internal to that templ file.

* Be mindful of `templ` language rules, and avoid `html/template`-isms (eg $prefixed variables).

* DO NOT use any extra library (frontend or backend) even if documentation or other sources recommend so. Our frontend is just HTML and Bootstrap. We have no plans to pull in HTMX or such.

* Use raw go statements `{{ }}` sparingly in templates, only to do variable assignments. 

* Do not use complex to understand features like fragments. Use only functional composition.

## Appendix A: Service methods with return types
```
DictionaryService:
  * func (s *DictionaryService) GetEntries(ctx context.Context, dictionaryName string, words []string, tl common.Transliteration) (map[string]DictionaryEntry, error)
  * func (s *DictionaryService) Suggest(ctx context.Context, dictName string, partialWord string, tl common.Transliteration) (Suggestions, error)
  * func (s *DictionaryService) Search(ctx context.Context, dictionaryName string, searchParams SearchParams) (SearchResults, error)
  * func (s *DictionaryService) Related(ctx context.Context, dictName string, word string) (SearchResults, error)

ExcerptService:
  * func (s *ExcerptService) Get(ctx context.Context, paths []QualifiedPath) (*ExcerptTemplateData, error)
  * func (s *ExcerptService) Search(ctx context.Context, search SearchParams) (*ExcerptSearchData, error)
  * func (s *ExcerptService) GetHier(ctx context.Context, scriptureName string, path []int) (*Hierarchy, error)
```