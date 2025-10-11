#!/usr/bin/env python3
import tomllib
import json
import argparse
from pathlib import Path


def load_toml(filepath):
    """Load TOML file and return parsed data."""
    with open(filepath, "rb") as f:
        return tomllib.load(f)


def generate_mappings(data_dir):
    """Generate SLP1 -> IAST, HK, and Devanagari mappings."""

    # Load all schemes
    slp1_data = load_toml(data_dir / "slp1.toml")
    iast_data = load_toml(data_dir / "iast.toml")
    hk_data = load_toml(data_dir / "hk.toml")

    # Create forward mappings (devanagari -> scheme) by category
    dev_to_iast = {}
    dev_to_hk = {}

    for section_name, section in iast_data.items():
        if isinstance(section, dict):
            dev_to_iast.update(section)

    for section_name, section in hk_data.items():
        if isinstance(section, dict):
            dev_to_hk.update(section)

    # Generate SLP1 -> target mappings with categories
    slp1_to_iast = {}
    slp1_to_hk = {}
    slp1_to_devanagari = {}

    unmapped = {"iast": [], "hk": [], "devanagari": []}

    # Process each category from SLP1
    for category, section in slp1_data.items():
        if not isinstance(section, dict):
            continue

        slp1_to_iast[category] = {}
        slp1_to_hk[category] = {}
        slp1_to_devanagari[category] = {}

        for dev_char, slp1_char in section.items():
            # SLP1 -> Devanagari (direct)
            slp1_to_devanagari[category][slp1_char] = dev_char

            # SLP1 -> IAST (via Devanagari)
            if dev_char in dev_to_iast:
                slp1_to_iast[category][slp1_char] = dev_to_iast[dev_char]
            else:
                unmapped["iast"].append(
                    {
                        "category": category,
                        "slp1": slp1_char,
                        "devanagari": dev_char,
                        "reason": "no_iast_equivalent",
                    }
                )

            # SLP1 -> HK (via Devanagari)
            if dev_char in dev_to_hk:
                slp1_to_hk[category][slp1_char] = dev_to_hk[dev_char]
            else:
                unmapped["hk"].append(
                    {
                        "category": category,
                        "slp1": slp1_char,
                        "devanagari": dev_char,
                        "reason": "no_hk_equivalent",
                    }
                )

    # Count total mappings
    total_iast = sum(len(cat) for cat in slp1_to_iast.values())
    total_hk = sum(len(cat) for cat in slp1_to_hk.values())
    total_dev = sum(len(cat) for cat in slp1_to_devanagari.values())

    result = {
        "mappings": {
            "slp1_to_iast": slp1_to_iast,
            "slp1_to_hk": slp1_to_hk,
            "slp1_to_devanagari": slp1_to_devanagari,
        },
        "unmapped": unmapped,
        "stats": {
            "slp1_to_iast_count": total_iast,
            "slp1_to_hk_count": total_hk,
            "slp1_to_devanagari_count": total_dev,
            "unmapped_iast": len(unmapped["iast"]),
            "unmapped_hk": len(unmapped["hk"]),
            "unmapped_devanagari": len(unmapped["devanagari"]),
        },
    }

    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate SLP1 transliteration mappings"
    )
    parser.add_argument(
        "--data-dir", required=True, help="Directory containing TOML files"
    )
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    result = generate_mappings(data_dir)

    # Print only stats to stdout
    print(json.dumps(result["stats"], indent=2))

    # Save full result to file
    output_file = data_dir / "slp1_mappings.json"
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"\nFull mappings saved to: {output_file}")
