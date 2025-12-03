## Feature: related verses based on vector similarity

We store verses of a scripture in a JSONL file. We need to write a python script `script/cosine_similarity.py` which does the following:

* take arguments using argparse, with clear help texts
  --input-file (JSONL file where each line represents an `Excerpt` struct)
  --embedding-model (required, string)
  --batch-size (int, default=32) number of embeddings to generate in one batch
  --auxiliaries (multi, require 1)
  --threshold (optional) # Minimum cosine similarity threshold, Otherwise pick 5 highest scores no matter how low they are
  --tei-endpoint (huggingface TextEmbeddingInference (TEI) container endpoint instead of running model locally - if this is provided, do not import any HF library locally).
  --output-file (JSONL file to store verse index -> [similar verse indices] mapping.

* Load embedding models from hugging face based on model passed. (Or use TEI REST API without importing HF libraries if TEI REST endpoint is passed)
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

## Integration with Dhee web application
* We adapt this embedding data during `preprocess` step since it's a relatively costly affair.
* Update the preprocess part `parse_rv_tei.go` (which is called from `main`) to load the embeddings JSONL as a Map into the memory and add the related verses to the excerpt structure in `app/excerpts/schema.go`, which ultimately goes into the preprocessed JSONL file.
* After this, everything will be seemlessly integrated into preprocess -> index -> serve flow.

## UI
* The excerpt will be passed to `excerpts.html` (passing through `layout.html` which would set bootstrap css etc..)
* In excerpts.html, at the end of all cards add a card called related.
* create a bootstrap secondary badge for each related verse and link it like `../{Path}`. Upon hover it should show the Cosine similarity score.

## Other guidelines for Python
* type-annotate the python function signatures and return types, as well as any complex collection types.
* There should be a `main()` function in python script. (i.e do not write the procedure in top level script).

## Appendix 1: snippet from huggingface TEI endpoints

```python
embedding_input = 'This input will get multiplied' * 10000
print(f'The length of the embedding_input is: {len(embedding_input)}')
response = endpoint.client.post(json={"inputs": embedding_input, 'truncate': True}, task="feature-extraction")
response = np.array(json.loads(response.decode()))
response[0][:20]
```
