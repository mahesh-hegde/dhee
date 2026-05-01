#!/usr/bin/env python3
import argparse
import json
import logging
import os
import time
from typing import Any, Dict, List

import numpy as np

MAX_RELATED_EXCERPTS = 5
MAX_CANDIDATES_FOR_RERANKING = 50


def main() -> None:
    """Main function to run the script."""
    logging.basicConfig(
        level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
    )

    parser = argparse.ArgumentParser(
        description="Compute cosine similarity for verses based on embeddings."
    )
    parser.add_argument("--input-file", required=True, help="JSONL file with excerpts.")
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
        "--batch-cooldown",
        type=float,
        default=0,
        help="Cooldown in seconds after each embedding batch to reduce CPU/GPU heat.",
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
        help=f"Minimum similarity threshold. If not provided, top {MAX_RELATED_EXCERPTS} are picked.",
    )
    parser.add_argument(
        "--tei-endpoint",
        help="Hugging Face TEI container endpoint. If provided, no local HF import.",
    )
    parser.add_argument(
        "--reranker-model",
        help="Hugging Face cross-encoder model for reranking. If specified, reranks top 50 candidates.",
    )
    parser.add_argument(
        "--query-prefix",
        default="",
        help="Prefix to add to query text for reranker (e.g., 'WHICH VERSES ARE SIMILAR TO THIS VERSE?').",
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

    # Initialize reranker if specified
    reranker = None
    if args.reranker_model:
        from sentence_transformers import CrossEncoder

        logging.info(f"Loading reranker model: {args.reranker_model}")
        reranker = CrossEncoder(args.reranker_model)

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

        logging.info(
            f"Found {len(texts_to_embed)} texts to embed for auxiliary '{aux}'."
        )

        embeddings = get_embeddings(texts_to_embed)
        embeddings = embeddings / np.linalg.norm(embeddings, axis=1, keepdims=True)

        logging.info("Embeddings generated. Calculating similarities...")

        # Determine number of candidates to retrieve
        num_candidates = MAX_CANDIDATES_FOR_RERANKING + 1 if reranker else MAX_RELATED_EXCERPTS + 1

        for i, source_index in enumerate(indices_for_texts):
            source_embedding = embeddings[i]
            # Cosine similarity is dot product of normalized vectors
            similarities = np.dot(embeddings, source_embedding)
            pick = -num_candidates
            # Get top candidates to exclude self later
            top_indices = np.argpartition(similarities, pick)[pick:]
            # Sort these top indices by similarity
            top_indices = top_indices[np.argsort(similarities[top_indices])][::-1]

            related_excerpts: List[Dict[str, Any]] = []

            if reranker:
                # Reranking mode: collect candidates and rerank them
                candidates_data: List[tuple] = []  # (index_in_top_indices, target_index, embedding_score)

                for j in top_indices:
                    if i == j:
                        continue  # Skip self

                    target_index = indices_for_texts[j]
                    embedding_score = float(similarities[j])
                    candidates_data.append((j, target_index, embedding_score))

                if candidates_data:
                    # Build query and candidate pairs for reranking
                    query_text = args.query_prefix + " " if args.query_prefix else ""
                    query_text += " ".join(excerpt["auxiliaries"][aux].get("text", []))

                    pairs = [
                        (query_text, " ".join(excerpts_by_index[target_idx]["auxiliaries"][aux].get("text", [])))
                        for _, target_idx, _ in candidates_data
                    ]

                    # Get reranker scores
                    reranker_scores = reranker.predict(pairs)

                    # Build results with reranker scores
                    for score_idx, (_, target_index, _) in enumerate(candidates_data):
                        score = float(reranker_scores[score_idx])

                        if args.threshold and score < args.threshold:
                            continue

                        related_excerpts.append(
                            {"readable_index": target_index, "score": score}
                        )

                        if len(related_excerpts) >= MAX_RELATED_EXCERPTS:
                            break

                    # Sort by reranker score descending
                    related_excerpts.sort(key=lambda x: x["score"], reverse=True)
            else:
                # Embedding-only mode: use original logic
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

                    if len(related_excerpts) >= MAX_RELATED_EXCERPTS:
                        break

            if source_index not in all_related:
                all_related[source_index] = []

            existing_related = {
                item["readable_index"]: item for item in all_related[source_index]
            }
            for new_item in related_excerpts:
                if new_item["readable_index"] in existing_related:
                    if (
                        new_item["score"]
                        > existing_related[new_item["readable_index"]]["score"]
                    ):
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
            # Sort by score descending and take top MAX_RELATED_EXCERPTS across all auxiliaries
            related_list.sort(key=lambda x: x["score"], reverse=True)
            output_data = {
                "readable_index": readable_index,
                "related": related_list[:MAX_RELATED_EXCERPTS],
            }
            f.write(json.dumps(output_data) + "\n")


if __name__ == "__main__":
    main()


## Textual similarity logic
## For each word in verse A
##    wf = count of word in all verses
##    wlf = count of lemma in all verses
##    wscore = (1 if word in verse_B else 0) / wf
##    wscore += (1 if word_lemma in verse_B else 0) / wlf
##    if previous word position +1 == current word position
## 	    score += bonus
##      bonus += 1
##    else
##      bonus = wscore
## final_score = score / (len(verse_A_words) + len(verse_B_words))
