## Feature: related verses based on vector similarity

We store verses of a scripture in a JSONL file. We need to write a python script which does the following

* take arguments for --embedding-model (required, string), --auxiliaries (multi, require 1), --threshold (optional)

* Load embedding models from hugging face based on model passed.
* Load all verses into memory from given file
* create a map verse (readable index -> verse data) (we will need it later)
* for each auxiliary
  * create a map (verse -> embedding vector)
  * for each verse loaded from JSONL file
    * compute embedding of that verse if the auxiliary by given key is present
  * for each verse above which has embeddings successfully generated
    * compute the closest 5 verses by looping over the whole map (O(n^2) time complexity is fine for now because its a preprocess script).
    * compute cosine similarity and store them as `related` field in the parent map consisting of verse data.
    * if --threshold is provided, we should store the verse only if similarity is above threshold
* write the output as <file_name_without_jsonl>.emb.jsonl

We need to do this in python because calling huggingface models from python is easier than its in go.

## UI
* The excerpt will be passed to `excerpts.html` (passing through `layout.html` which would set bootstrap css etc..)
* In excerpts.html, at the end of all cards add a card called related.
* create a bootstrap secondary badge for each related verse and link it like `../{Path}`.

## Other guidelines
* type-annotate the python function signatures and return types, as well as any complex collection types.
* There should be a `main()` function in python script. (i.e do not write the procedure in top level script).