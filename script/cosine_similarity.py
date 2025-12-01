#!/usr/bin/env python3
import argparse
import json
import logging
import os
from typing import Any, Dict, List

import numpy as np


def main() -> None:
	"""Main function to run the script."""
	logging.basicConfig(
		level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
	)

	parser = argparse.ArgumentParser(
		description="Compute cosine similarity for verses based on embeddings."
	)
	parser.add_argument(
		"--input-file", required=True, help="JSONL file with excerpts."
	)
	parser.add_argument(
		"--embedding-model",
		help="Name of the Hugging Face sentence embedding model.",
	)
	parser.add_argument(
		"--batch-size",
		type=int,
		default=32,
		help="Number of embeddings to generate in one batch.",
	)
	parser.add_argument(
		"--auxiliaries",
		nargs="+",
		required=True,
		help="One or more auxiliary keys to use for embedding.",
	)
	parser.add_argument(
		"--threshold",
		type=float,
		help="Minimum cosine similarity threshold. If not provided, top 5 are picked.",
	)
	parser.add_argument(
		"--tei-endpoint",
		help="Hugging Face TEI container endpoint. If provided, no local HF import.",
	)
	parser.add_argument("--output-file", help="JSONL file to store the output.")

	args = parser.parse_args()

	if not args.embedding_model and not args.tei_endpoint:
		parser.error("Either --embedding-model or --tei-endpoint is required.")

	# Load excerpts
	excerpts: List[Dict[str, Any]] = []
	with open(args.input_file, "r", encoding="utf-8") as f:
		for line in f:
			excerpts.append(json.loads(line))

	logging.info(f"Loaded {len(excerpts)} excerpts.")

	excerpts_by_index: Dict[str, Dict[str, Any]] = {
		e["readable_index"]: e for e in excerpts
	}

	if args.tei_endpoint:
		import requests
		from requests.adapters import HTTPAdapter
		from urllib3.util.retry import Retry

		session = requests.Session()
		retries = Retry(total=5, backoff_factor=1, status_forcelist=[502, 503, 504])
		session.mount("http://", HTTPAdapter(max_retries=retries))
		session.mount("https://", HTTPAdapter(max_retries=retries))

		def get_embeddings_tei(texts: List[str]) -> np.ndarray:
			response = session.post(
				args.tei_endpoint, json={"inputs": texts, "truncate": True}
			)
			response.raise_for_status()
			return np.array(response.json())

		get_embeddings = get_embeddings_tei
	else:
		from sentence_transformers import SentenceTransformer

		model = SentenceTransformer(args.embedding_model)

		def get_embeddings_local(texts: List[str]) -> np.ndarray:
			return model.encode(
				texts, batch_size=args.batch_size, show_progress_bar=True
			)

		get_embeddings = get_embeddings_local

	all_related: Dict[str, List[Dict[str, Any]]] = {}

	for aux in args.auxiliaries:
		logging.info(f"Processing auxiliary: {aux}")

		texts_to_embed: List[str] = []
		indices_for_texts: List[str] = []

		for index, excerpt in excerpts_by_index.items():
			if "auxiliaries" in excerpt and aux in excerpt["auxiliaries"]:
				text = " ".join(excerpt["auxiliaries"][aux].get("text", []))
				if text:
					texts_to_embed.append(text)
					indices_for_texts.append(index)

		if not texts_to_embed:
			logging.info(f"No texts found for auxiliary '{aux}'. Skipping.")
			continue

		logging.info(f"Found {len(texts_to_embed)} texts to embed for auxiliary '{aux}'.")

		embeddings = get_embeddings(texts_to_embed)
		embeddings = embeddings / np.linalg.norm(embeddings, axis=1, keepdims=True)

		logging.info("Embeddings generated. Calculating similarities...")

		for i, source_index in enumerate(indices_for_texts):
			source_embedding = embeddings[i]
			# Cosine similarity is dot product of normalized vectors
			similarities = np.dot(embeddings, source_embedding)

			# Get top 6 indices to exclude self later
			top_indices = np.argpartition(similarities, -6)[-6:]
			# Sort these top indices by similarity
			top_indices = top_indices[np.argsort(similarities[top_indices])][::-1]

			related_excerpts: List[Dict[str, Any]] = []
			for j in top_indices:
				if i == j:
					continue  # Skip self

				target_index = indices_for_texts[j]
				score = float(similarities[j])

				if args.threshold and score < args.threshold:
					continue

				related_excerpts.append(
					{"readable_index": target_index, "score": score}
				)

				if len(related_excerpts) >= 5:
					break

			if source_index not in all_related:
				all_related[source_index] = []

			existing_related = {
				item["readable_index"]: item for item in all_related[source_index]
			}
			for new_item in related_excerpts:
				if new_item["readable_index"] in existing_related:
					if new_item["score"] > existing_related[new_item["readable_index"]]["score"]:
						existing_related[new_item["readable_index"]] = new_item
				else:
					existing_related[new_item["readable_index"]] = new_item

			all_related[source_index] = list(existing_related.values())

	output_file = args.output_file
	if not output_file:
		base, _ = os.path.splitext(args.input_file)
		output_file = f"{base}.emb.jsonl"

	logging.info(f"Writing output to {output_file}")
	with open(output_file, "w", encoding="utf-8") as f:
		for readable_index, related_list in all_related.items():
			# Sort by score descending and take top 5 across all auxiliaries
			related_list.sort(key=lambda x: x["score"], reverse=True)
			output_data = {"readable_index": readable_index, "related": related_list[:5]}
			f.write(json.dumps(output_data) + "\n")


if __name__ == "__main__":
	main()
