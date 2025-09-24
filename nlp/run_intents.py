#!/usr/bin/env python3
"""
NLP Intent to QoS Mapping Module

This module processes natural language intents and maps them to QoS parameters
for O-RAN network slice configuration. It validates output against schema.json
and outputs results in JSONL format.

Usage:
    python run_intents.py <intent_file> [options]

Example:
    python run_intents.py fixtures/intents.txt --output results.jsonl
"""

import argparse
import json
import logging
import re
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional, TextIO, Tuple, cast

import jsonschema

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class IntentToQoSMapper:
    """Maps natural language intents to QoS parameters for network slices."""

    def __init__(self, schema_path: str = "schema.json"):
        """Initialize the mapper with QoS schema validation."""
        self.schema_path = Path(schema_path)
        self.schema = self._load_schema()

        # Intent mapping patterns
        self.intent_patterns: Dict[str, Dict[str, Any]] = {
            # eMBB patterns - high bandwidth
            "embb": {
                "keywords": [
                    "high bandwidth",
                    "embb",
                    "video streaming",
                    "multimedia",
                    "content delivery",
                    "4k video",
                    "streaming",
                    "download",
                    "throughput",
                    "ar",
                    "vr",
                    "augmented reality",
                    "virtual reality",
                ],
                "qos": {"bandwidth": 5, "latency": 9, "slice_type": "eMBB"},
            },
            # uRLLC patterns - low latency
            "urllc": {
                "keywords": [
                    "low latency",
                    "urllc",
                    "real-time",
                    "mission critical",
                    "industrial",
                    "emergency",
                    "control",
                    "robotics",
                    "automation",
                    "ultra-low",
                    "critical",
                    "responsive",
                    "immediate",
                ],
                "qos": {"bandwidth": 1, "latency": 1, "slice_type": "uRLLC"},
            },
            # Balanced patterns - general purpose
            "balanced": {
                "keywords": [
                    "balanced",
                    "general",
                    "standard",
                    "typical",
                    "normal",
                    "enterprise",
                    "business",
                    "general purpose",
                    "medium",
                    "moderate",
                    "average",
                    "default",
                ],
                "qos": {"bandwidth": 3, "latency": 9, "slice_type": "balanced"},
            },
        }

    def _load_schema(self) -> Dict[str, Any]:
        """Load and return the QoS JSON schema."""
        try:
            if not self.schema_path.exists():
                raise FileNotFoundError(f"Schema file not found: {self.schema_path}")

            with open(self.schema_path, "r", encoding="utf-8") as f:
                schema = json.load(f)

            logger.info("Loaded QoS schema from %s", self.schema_path)
            return schema

        except Exception as e:
            logger.error("Failed to load schema: %s", e)
            raise

    def _normalize_intent(self, intent: str) -> str:
        """Normalize intent text for pattern matching."""
        # Convert to lowercase and clean whitespace
        normalized = re.sub(r"\s+", " ", intent.lower().strip())
        # Remove common punctuation
        normalized = re.sub(r"[.,!?;:]", "", normalized)
        return normalized

    def _match_intent_pattern(self, intent: str) -> Tuple[str, Dict[str, Any]]:
        """Match intent against known patterns and return QoS parameters."""
        normalized_intent = self._normalize_intent(intent)

        # Score each pattern based on keyword matches
        pattern_scores: Dict[str, Dict[str, Any]] = {}

        for pattern_name, pattern_data in self.intent_patterns.items():
            score = 0
            keywords_found = []

            for keyword in pattern_data["keywords"]:
                if keyword.lower() in normalized_intent:
                    score += 1
                    keywords_found.append(keyword)

            if score > 0:
                pattern_scores[pattern_name] = {
                    "score": score,
                    "keywords": keywords_found,
                    "qos": (
                        dict(pattern_data["qos"])
                        if isinstance(pattern_data["qos"], dict)
                        else {}
                    ),
                }

        # Return the highest scoring pattern
        if pattern_scores:
            best_pattern = max(
                pattern_scores.keys(), key=lambda k: int(pattern_scores[k]["score"])
            )
            logger.debug(
                f"Matched pattern '{best_pattern}' with keywords: "
                f"{pattern_scores[best_pattern]['keywords']}"
            )
            return best_pattern, cast(
                Dict[str, Any], pattern_scores[best_pattern]["qos"]
            )

        # Default to balanced if no patterns match
        logger.warning(
            f"No pattern matched for intent: '{intent}'. Using balanced profile."
        )
        return "balanced", cast(
            Dict[str, Any], self.intent_patterns["balanced"]["qos"].copy()
        )

    def _enhance_qos_parameters(
        self, qos: Dict[str, Any], intent: str
    ) -> Dict[str, Any]:
        """Enhance QoS parameters with additional context-specific values."""
        normalized_intent = self._normalize_intent(intent)

        # Add reliability for critical applications
        if any(
            keyword in normalized_intent
            for keyword in ["critical", "emergency", "mission", "industrial"]
        ):
            qos["reliability"] = 99.99

        # Add jitter constraints for real-time applications
        if any(
            keyword in normalized_intent
            for keyword in ["real-time", "gaming", "voice", "video call"]
        ):
            qos["jitter"] = 1.0

        # Add packet loss constraints for quality-sensitive apps
        if any(
            keyword in normalized_intent
            for keyword in ["streaming", "video", "voice", "multimedia"]
        ):
            qos["packet_loss"] = 0.1

        return qos

    def map_intent_to_qos(self, intent: str) -> Dict[str, Any]:
        """Map a single intent to QoS parameters."""
        try:
            # Match intent to pattern
            pattern_name, qos = self._match_intent_pattern(intent)

            # Enhance with additional parameters
            qos = self._enhance_qos_parameters(qos, intent)

            # Validate against schema
            self._validate_qos(qos)

            logger.info(
                "Mapped intent '%s...' to %s profile", intent[:50], pattern_name
            )
            return qos

        except Exception as e:
            logger.error("Failed to map intent '%s': %s", intent, e)
            raise

    def _validate_qos(self, qos: Dict[str, Any]) -> None:
        """Validate QoS parameters against the schema."""
        try:
            jsonschema.validate(instance=qos, schema=self.schema)
            logger.debug("QoS parameters validated successfully: %s", qos)

        except jsonschema.ValidationError as e:
            error_msg = f"QoS validation error: {e.message}"
            logger.error(error_msg)
            raise ValueError(error_msg) from e

        except Exception as e:
            logger.error("Unexpected validation error: %s", e)
            raise

    def process_intent_file(self, input_file: str) -> List[Dict[str, Any]]:
        """Process multiple intents from a file."""
        input_path = Path(input_file)

        if not input_path.exists():
            raise FileNotFoundError(f"Intent file not found: {input_file}")

        results = []

        try:
            with open(input_path, "r", encoding="utf-8") as f:
                lines = f.readlines()

            logger.info("Processing %d intents from %s", len(lines), input_file)

            for line_num, line in enumerate(lines, 1):
                line = line.strip()
                if not line or line.startswith("#"):  # Skip empty lines and comments
                    continue

                try:
                    qos = self.map_intent_to_qos(line)
                    results.append({"intent": line, "qos": qos, "line": line_num})

                except (ValueError, KeyError, json.JSONDecodeError) as e:
                    logger.warning("Failed to process line %d: %s", line_num, e)
                    # Add error entry
                    results.append({"intent": line, "error": str(e), "line": line_num})

            success_count = len([r for r in results if "qos" in r])
            logger.info("Successfully processed %d intents", success_count)
            return results

        except Exception as e:
            logger.error("Failed to process intent file: %s", e)
            raise


def write_jsonl_output(
    results: List[Dict[str, Any]], output_file: Optional[str] = None
) -> None:
    """Write results to JSONL format (one JSON object per line)."""
    file_handle: TextIO
    if output_file:
        output_path = Path(output_file)
        logger.info("Writing results to %s", output_file)
        file_handle = open(output_path, "w", encoding="utf-8")
    else:
        file_handle = sys.stdout

    try:
        for result in results:
            if "qos" in result:
                # Output successful QoS mapping
                output_data = result["qos"]
            else:
                # Output error information
                output_data = {
                    "error": result.get("error", "Unknown error"),
                    "intent": result["intent"],
                }

            json.dump(output_data, file_handle, separators=(",", ":"))
            file_handle.write("\n")

        if output_file:
            logger.info("Results written to %s", output_file)

    finally:
        if output_file:
            file_handle.close()


def main():
    """Main entry point for the intent-to-QoS mapping CLI."""
    parser = argparse.ArgumentParser(
        description="Map natural language intents to QoS parameters for O-RAN "
        "network slices"
    )

    parser.add_argument(
        "intent_file",
        help="Input file containing natural language intents (one per line)",
    )

    parser.add_argument(
        "-o", "--output", help="Output file for JSONL results (default: stdout)"
    )

    parser.add_argument(
        "-s",
        "--schema",
        default="schema.json",
        help="Path to QoS schema file (default: schema.json)",
    )

    parser.add_argument(
        "-v", "--verbose", action="store_true", help="Enable verbose logging"
    )

    parser.add_argument(
        "--validate-only",
        action="store_true",
        help="Only validate intents without processing",
    )

    args = parser.parse_args()

    # Configure logging level
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    try:
        # Initialize the mapper
        mapper = IntentToQoSMapper(schema_path=args.schema)

        # Process the intent file
        results = mapper.process_intent_file(args.intent_file)

        if args.validate_only:
            # Just report validation results
            success_count = len([r for r in results if "qos" in r])
            error_count = len(results) - success_count

            print(
                f"Validation complete: {success_count} successful, {error_count} errors"
            )

            if error_count > 0:
                print("\nErrors:")
                for result in results:
                    if "error" in result:
                        print(f"Line {result['line']}: {result['error']}")
                sys.exit(1)
        else:
            # Output JSONL results
            write_jsonl_output(results, args.output)

        logger.info("Intent-to-QoS mapping completed successfully")

    except Exception as e:
        logger.error("Application error: %s", e)
        sys.exit(1)


if __name__ == "__main__":
    main()
