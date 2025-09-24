"""
Comprehensive unit tests for the NLP Intent Parser module.
Tests cover intent parsing, QoS mapping, validation, and edge cases.
Target: >90% code coverage with production-grade test scenarios.
"""

import os
import sys
from unittest.mock import Mock, patch

import pytest

# Add the parent directory to the path to import the module
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "../../")))

from intent_parser import IntentParser, IntentValidationError  # noqa: E402
from schema_validator import SchemaValidator  # noqa: E402


class TestIntentParser:
    """Test suite for the IntentParser class with comprehensive coverage."""

    @pytest.fixture
    def parser(self):
        """Create a fresh IntentParser instance for each test."""
        return IntentParser()

    @pytest.fixture
    def schema_validator(self):
        """Create a mock schema validator."""
        validator = Mock(spec=SchemaValidator)
        validator.validate.return_value = True
        return validator

    @pytest.fixture
    def sample_intents(self):
        """Provide sample intent texts for testing."""
        return {
            "high_priority": "Deploy a high-priority 5G network slice for emergency services with ultra-low latency",
            "video_streaming": "Create a network slice for 4K video streaming with high bandwidth requirements",
            "iot_basic": "Set up a basic IoT network slice for sensor data collection",
            "autonomous_vehicles": "Establish an ultra-reliable low-latency slice for autonomous vehicle communication",
            "enterprise": "Deploy an enterprise network slice with guaranteed bandwidth and moderate latency",
            "gaming": "Create a gaming network slice with low latency and high throughput",
            "machine_learning": "Set up a high-bandwidth slice for machine learning model training",
            "voice_calls": "Deploy a voice communication slice with consistent quality",
            "file_transfer": "Create a bulk data transfer slice with maximum throughput",
            "critical_infrastructure": "Establish a mission-critical slice for power grid monitoring",
        }

    @pytest.fixture
    def expected_qos_mappings(self):
        """Expected QoS mappings for validation."""
        return {
            "high_priority": {
                "priority": "high",
                "latency_ms": 1,
                "throughput_mbps": 100,
                "reliability": 99.999,
                "slice_type": "urllc",
            },
            "video_streaming": {
                "priority": "medium",
                "latency_ms": 10,
                "throughput_mbps": 50,
                "reliability": 99.9,
                "slice_type": "embb",
            },
            "iot_basic": {
                "priority": "low",
                "latency_ms": 100,
                "throughput_mbps": 1,
                "reliability": 99.0,
                "slice_type": "mmtc",
            },
        }

    def test_parser_initialization(self, parser):
        """Test that parser initializes correctly."""
        assert parser is not None
        assert hasattr(parser, "parse_intent")
        assert hasattr(parser, "map_to_qos")
        assert hasattr(parser, "validate_intent")

    def test_basic_intent_parsing(self, parser, sample_intents):
        """Test basic intent parsing functionality."""
        intent = sample_intents["high_priority"]
        result = parser.parse_intent(intent)

        assert result is not None
        assert "intent_type" in result
        assert "keywords" in result
        assert "confidence" in result
        assert result["confidence"] > 0.5

    def test_qos_mapping_high_priority(
        self, parser, sample_intents, expected_qos_mappings
    ):
        """Test QoS mapping for high-priority intents."""
        intent = sample_intents["high_priority"]
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        expected = expected_qos_mappings["high_priority"]
        assert qos["priority"] == expected["priority"]
        assert qos["latency_ms"] <= expected["latency_ms"]
        assert qos["throughput_mbps"] >= expected["throughput_mbps"]
        assert qos["reliability"] >= expected["reliability"]

    def test_qos_mapping_video_streaming(
        self, parser, sample_intents, expected_qos_mappings
    ):
        """Test QoS mapping for video streaming intents."""
        intent = sample_intents["video_streaming"]
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        expected = expected_qos_mappings["video_streaming"]
        assert qos["slice_type"] == expected["slice_type"]
        assert qos["throughput_mbps"] >= expected["throughput_mbps"]

    def test_qos_mapping_iot_basic(self, parser, sample_intents, expected_qos_mappings):
        """Test QoS mapping for basic IoT intents."""
        intent = sample_intents["iot_basic"]
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        expected = expected_qos_mappings["iot_basic"]
        assert qos["slice_type"] == expected["slice_type"]
        assert qos["priority"] == expected["priority"]

    @pytest.mark.parametrize(
        "intent_key,expected_slice_type",
        [
            ("autonomous_vehicles", "urllc"),
            ("enterprise", "embb"),
            ("gaming", "urllc"),
            ("machine_learning", "embb"),
            ("voice_calls", "urllc"),
            ("file_transfer", "embb"),
            ("critical_infrastructure", "urllc"),
        ],
    )
    def test_slice_type_classification(
        self, parser, sample_intents, intent_key, expected_slice_type
    ):
        """Test slice type classification for various intents."""
        intent = sample_intents[intent_key]
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        assert qos["slice_type"] == expected_slice_type

    def test_intent_validation_valid_intent(self, parser, sample_intents):
        """Test validation of valid intents."""
        intent = sample_intents["high_priority"]
        parsed = parser.parse_intent(intent)

        assert parser.validate_intent(parsed) is True

    def test_intent_validation_invalid_intent(self, parser):
        """Test validation of invalid intents."""
        invalid_intent = {"invalid": "structure"}

        with pytest.raises(IntentValidationError):
            parser.validate_intent(invalid_intent)

    def test_empty_intent_handling(self, parser):
        """Test handling of empty or None intents."""
        with pytest.raises(ValueError):
            parser.parse_intent("")

        with pytest.raises(ValueError):
            parser.parse_intent(None)

    def test_malformed_intent_handling(self, parser):
        """Test handling of malformed intents."""
        malformed_intents = [
            "   ",  # Only whitespace
            "a" * 1000,  # Too long
            "!@#$%^&*()",  # Only special characters
            "123456789",  # Only numbers
        ]

        for intent in malformed_intents:
            with pytest.raises((ValueError, IntentValidationError)):
                result = parser.parse_intent(intent)
                parser.validate_intent(result)

    def test_keyword_extraction(self, parser):
        """Test keyword extraction from intents."""
        intent = "Deploy a high-priority 5G network slice with ultra-low latency for emergency services"
        parsed = parser.parse_intent(intent)

        expected_keywords = ["high-priority", "5G", "ultra-low", "latency", "emergency"]
        keywords = parsed.get("keywords", [])

        # Check that some expected keywords are found
        found_keywords = [
            kw
            for kw in expected_keywords
            if any(kw.lower() in k.lower() for k in keywords)
        ]
        assert len(found_keywords) >= 2

    def test_confidence_scoring(self, parser, sample_intents):
        """Test confidence scoring for different intent types."""
        # Well-defined intents should have high confidence
        high_confidence_intent = sample_intents["autonomous_vehicles"]
        parsed_high = parser.parse_intent(high_confidence_intent)
        assert parsed_high["confidence"] > 0.8

        # Ambiguous intents should have lower confidence
        ambiguous_intent = "create something for networking"
        parsed_low = parser.parse_intent(ambiguous_intent)
        assert parsed_low["confidence"] < 0.7

    def test_qos_parameter_boundaries(self, parser):
        """Test QoS parameter boundary conditions."""
        # Test minimum latency requirements
        ultra_low_latency_intent = "ultra-low latency critical application"
        parsed = parser.parse_intent(ultra_low_latency_intent)
        qos = parser.map_to_qos(parsed)
        assert qos["latency_ms"] <= 5

        # Test maximum throughput requirements
        high_bandwidth_intent = "maximum bandwidth high-speed data transfer"
        parsed = parser.parse_intent(high_bandwidth_intent)
        qos = parser.map_to_qos(parsed)
        assert qos["throughput_mbps"] >= 100

    def test_json_schema_compliance(self, parser, sample_intents):
        """Test that output complies with expected JSON schema."""
        intent = sample_intents["enterprise"]
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        # Required fields
        required_fields = [
            "priority",
            "latency_ms",
            "throughput_mbps",
            "reliability",
            "slice_type",
        ]
        for field in required_fields:
            assert field in qos

        # Type validation
        assert isinstance(qos["priority"], str)
        assert isinstance(qos["latency_ms"], (int, float))
        assert isinstance(qos["throughput_mbps"], (int, float))
        assert isinstance(qos["reliability"], (int, float))
        assert isinstance(qos["slice_type"], str)

        # Value range validation
        assert qos["priority"] in ["low", "medium", "high"]
        assert 0.1 <= qos["latency_ms"] <= 1000
        assert 0.1 <= qos["throughput_mbps"] <= 1000
        assert 90.0 <= qos["reliability"] <= 99.999
        assert qos["slice_type"] in ["urllc", "embb", "mmtc"]

    def test_nlp_processor_integration(self, parser):
        """Test integration with NLP processor."""
        # Test that parser can handle high-priority emergency intents
        result = parser.parse_intent("emergency high-priority slice")
        assert result["confidence"] >= 0.8
        assert "emergency" in result.get("keywords", []) or "critical" in result.get(
            "keywords", []
        )
        assert result["slice_type"] == "urllc"

    def test_concurrent_parsing(self, parser, sample_intents):
        """Test concurrent intent parsing."""
        import concurrent.futures

        results = []
        errors = []

        def parse_intent_thread(intent_text):
            try:
                result = parser.parse_intent(intent_text)
                results.append(result)
            except Exception as e:
                errors.append(e)

        # Run multiple parsing operations concurrently
        with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
            futures = [
                executor.submit(parse_intent_thread, intent)
                for intent in sample_intents.values()
            ]
            concurrent.futures.wait(futures)

        assert len(errors) == 0, f"Concurrent parsing failed with errors: {errors}"
        assert len(results) == len(sample_intents)

    def test_performance_benchmark(self, parser, sample_intents):
        """Test performance benchmarks for intent parsing."""
        import time

        start_time = time.time()

        # Parse multiple intents
        for intent in sample_intents.values():
            parsed = parser.parse_intent(intent)
            _ = parser.map_to_qos(parsed)  # QoS mapping for performance test

        end_time = time.time()
        total_time = end_time - start_time
        avg_time_per_intent = total_time / len(sample_intents)

        # Performance requirement: each intent should be processed in <100ms
        assert (
            avg_time_per_intent < 0.1
        ), f"Average processing time {avg_time_per_intent}s exceeds 100ms threshold"

    def test_memory_usage(self, parser, sample_intents):
        """Test memory usage during intent parsing."""
        import os

        import psutil

        process = psutil.Process(os.getpid())
        initial_memory = process.memory_info().rss / 1024 / 1024  # MB

        # Process many intents
        for _ in range(100):
            for intent in sample_intents.values():
                parsed = parser.parse_intent(intent)
                _ = parser.map_to_qos(parsed)  # QoS mapping for memory test

        final_memory = process.memory_info().rss / 1024 / 1024  # MB
        memory_increase = final_memory - initial_memory

        # Memory increase should be reasonable (<50MB for 1000 operations)
        assert memory_increase < 50, f"Memory increase {memory_increase}MB is too high"

    def test_error_recovery(self, parser):
        """Test error recovery and graceful degradation."""
        # Test recovery from processing errors
        with patch.object(
            parser, "_extract_keywords", side_effect=Exception("NLP Error")
        ):
            try:
                result = parser.parse_intent("test intent")
                # Should still return a result with default values
                assert result is not None
                assert "confidence" in result
                assert result["confidence"] <= 0.5  # Lower confidence due to error
            except Exception:
                pytest.fail("Parser should handle NLP errors gracefully")

    def test_caching_functionality(self, parser):
        """Test intent parsing caching."""
        intent = "cached intent test"

        # Parse multiple times to test caching behavior
        result1 = parser.parse_intent(intent)
        result2 = parser.parse_intent(intent)
        result3 = parser.parse_intent(intent)

        # Results should be identical (showing consistent behavior)
        assert result1 == result2 == result3

        # Test that parser maintains consistent state
        assert result1["confidence"] == result2["confidence"]
        assert result1["slice_type"] == result2["slice_type"]

    def test_internationalization(self, parser):
        """Test handling of international characters and non-English text."""
        international_intents = [
            "创建高优先级网络切片",  # Chinese
            "créer une tranche réseau prioritaire",  # French
            "создать сетевой срез высокого приоритета",  # Russian
            "उच्च प्राथमिकता नेटवर्क स्लाइस बनाएं",  # Hindi
        ]

        for intent in international_intents:
            try:
                result = parser.parse_intent(intent)
                assert result is not None
                assert "confidence" in result
            except Exception as e:
                pytest.fail(
                    f"Failed to handle international text: {intent}, Error: {e}"
                )

    def test_thesis_performance_targets(self, parser, sample_intents):
        """Test that QoS mappings meet thesis performance targets."""
        # Thesis targets: {4.57, 2.77, 0.93} Mbps and {16.1, 15.7, 6.3} ms

        intents_to_test = [
            ("high_priority", {"min_throughput": 4.57, "max_latency": 6.3}),
            ("video_streaming", {"min_throughput": 2.77, "max_latency": 15.7}),
            ("iot_basic", {"min_throughput": 0.93, "max_latency": 16.1}),
        ]

        for intent_key, targets in intents_to_test:
            intent = sample_intents[intent_key]
            parsed = parser.parse_intent(intent)
            qos = parser.map_to_qos(parsed)

            assert (
                qos["throughput_mbps"] >= targets["min_throughput"]
            ), f"Throughput {qos['throughput_mbps']} < target {targets['min_throughput']} for {intent_key}"
            assert (
                qos["latency_ms"] <= targets["max_latency"]
            ), f"Latency {qos['latency_ms']} > target {targets['max_latency']} for {intent_key}"


class TestQoSMapping:
    """Test suite for QoS mapping functionality."""

    @pytest.fixture
    def parser(self):
        """Create an IntentParser instance for QoS mapping tests."""
        return IntentParser()

    def test_qos_mapping_initialization(self, parser):
        """Test QoS mapper initialization."""
        assert parser is not None
        assert hasattr(parser, "map_to_qos")

    def test_urllc_mapping(self, parser):
        """Test URLLC slice type mapping."""
        intent = "ultra-low latency critical communication"
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        assert qos["slice_type"] == "urllc"
        assert qos["latency_ms"] <= 5
        assert qos["reliability"] >= 99.99

    def test_embb_mapping(self, parser):
        """Test eMBB slice type mapping."""
        intent = "high bandwidth video streaming"
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        assert qos["slice_type"] == "embb"
        assert qos["throughput_mbps"] >= 50

    def test_mmtc_mapping(self, parser):
        """Test mMTC slice type mapping."""
        intent = "iot sensors massive connectivity"
        parsed = parser.parse_intent(intent)
        qos = parser.map_to_qos(parsed)

        assert qos["slice_type"] == "mmtc"
        assert qos["throughput_mbps"] <= 10


if __name__ == "__main__":
    # Run tests with coverage
    pytest.main(
        [
            __file__,
            "-v",
            "--cov=intent_parser",
            "--cov=schema_validator",
            "--cov-report=html",
            "--cov-report=term-missing",
            "--cov-fail-under=90",
        ]
    )
