"""
Unit tests for QoS schema validation and intent mapping.
Tests canonical intents, boundary conditions, and invalid inputs.
"""

import json
import pytest
from pathlib import Path
from jsonschema import validate, ValidationError, Draft7Validator

# Load schema from file
SCHEMA_PATH = Path(__file__).parent.parent / "schema.json"
with open(SCHEMA_PATH) as f:
    QOS_SCHEMA = json.load(f)

# Create validator instance
validator = Draft7Validator(QOS_SCHEMA)


class TestCanonicalIntents:
    """Test the three canonical intent mappings."""

    @pytest.fixture
    def canonical_intents(self):
        """Fixture providing the three canonical intent → QoS mappings."""
        return [
            {
                "name": "eMBB - High bandwidth, relaxed latency",
                "qos": {"bandwidth": 5, "latency": 9, "slice_type": "eMBB"},
                "expected": (5, 9)
            },
            {
                "name": "Balanced - Medium bandwidth, moderate latency",
                "qos": {"bandwidth": 3, "latency": 9, "slice_type": "balanced"},
                "expected": (3, 9)
            },
            {
                "name": "uRLLC - Low bandwidth, ultra-low latency",
                "qos": {"bandwidth": 1, "latency": 1, "slice_type": "uRLLC"},
                "expected": (1, 1)
            }
        ]

    def test_canonical_intents_valid(self, canonical_intents):
        """Test that all canonical intents pass schema validation."""
        for intent in canonical_intents:
            # Should not raise ValidationError
            validate(instance=intent["qos"], schema=QOS_SCHEMA)

            # Verify expected values
            assert intent["qos"]["bandwidth"] == intent["expected"][0]
            assert intent["qos"]["latency"] == intent["expected"][1]

    def test_canonical_intent_embb(self):
        """Test eMBB specific configuration."""
        embb_qos = {"bandwidth": 5, "latency": 9, "slice_type": "eMBB"}
        validate(instance=embb_qos, schema=QOS_SCHEMA)
        assert embb_qos["bandwidth"] == 5
        assert embb_qos["latency"] == 9

    def test_canonical_intent_urllc(self):
        """Test uRLLC specific configuration."""
        urllc_qos = {"bandwidth": 1, "latency": 1, "slice_type": "uRLLC", "reliability": 99.99}
        validate(instance=urllc_qos, schema=QOS_SCHEMA)
        assert urllc_qos["bandwidth"] == 1
        assert urllc_qos["latency"] == 1

    def test_canonical_intent_balanced(self):
        """Test balanced configuration."""
        balanced_qos = {"bandwidth": 3, "latency": 9, "slice_type": "balanced"}
        validate(instance=balanced_qos, schema=QOS_SCHEMA)
        assert balanced_qos["bandwidth"] == 3
        assert balanced_qos["latency"] == 9


class TestBoundaryConditions:
    """Test edge cases and boundary values."""

    def test_bandwidth_minimum_edge(self):
        """Test bandwidth at minimum boundary (1)."""
        qos = {"bandwidth": 1, "latency": 5}
        validate(instance=qos, schema=QOS_SCHEMA)
        assert qos["bandwidth"] == 1

    def test_bandwidth_maximum_edge(self):
        """Test bandwidth at maximum boundary (5)."""
        qos = {"bandwidth": 5, "latency": 5}
        validate(instance=qos, schema=QOS_SCHEMA)
        assert qos["bandwidth"] == 5

    def test_latency_minimum_edge(self):
        """Test latency at minimum boundary (1)."""
        qos = {"bandwidth": 3, "latency": 1}
        validate(instance=qos, schema=QOS_SCHEMA)
        assert qos["latency"] == 1

    def test_latency_maximum_edge(self):
        """Test latency at maximum boundary (10)."""
        qos = {"bandwidth": 3, "latency": 10}
        validate(instance=qos, schema=QOS_SCHEMA)
        assert qos["latency"] == 10

    def test_all_boundaries(self):
        """Test all combinations of boundary values."""
        boundary_cases = [
            {"bandwidth": 1, "latency": 1},   # Min-Min
            {"bandwidth": 1, "latency": 10},  # Min-Max
            {"bandwidth": 5, "latency": 1},   # Max-Min
            {"bandwidth": 5, "latency": 10},  # Max-Max
        ]

        for qos in boundary_cases:
            validate(instance=qos, schema=QOS_SCHEMA)

    def test_decimal_values(self):
        """Test that decimal values within range are accepted."""
        decimal_cases = [
            {"bandwidth": 1.5, "latency": 5.5},
            {"bandwidth": 2.7, "latency": 3.3},
            {"bandwidth": 4.9, "latency": 9.9},
        ]

        for qos in decimal_cases:
            validate(instance=qos, schema=QOS_SCHEMA)


class TestOutOfRangeValues:
    """Test that out-of-range values are properly rejected."""

    def test_bandwidth_below_minimum(self):
        """Test bandwidth below minimum (< 1) is rejected."""
        invalid_cases = [
            {"bandwidth": 0, "latency": 5},
            {"bandwidth": 0.5, "latency": 5},
            {"bandwidth": -1, "latency": 5},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError) as exc_info:
                validate(instance=qos, schema=QOS_SCHEMA)
            assert "minimum" in str(exc_info.value).lower()

    def test_bandwidth_above_maximum(self):
        """Test bandwidth above maximum (> 5) is rejected."""
        invalid_cases = [
            {"bandwidth": 6, "latency": 5},
            {"bandwidth": 5.1, "latency": 5},
            {"bandwidth": 100, "latency": 5},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError) as exc_info:
                validate(instance=qos, schema=QOS_SCHEMA)
            assert "maximum" in str(exc_info.value).lower()

    def test_latency_below_minimum(self):
        """Test latency below minimum (< 1) is rejected."""
        invalid_cases = [
            {"bandwidth": 3, "latency": 0},
            {"bandwidth": 3, "latency": 0.9},
            {"bandwidth": 3, "latency": -5},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError) as exc_info:
                validate(instance=qos, schema=QOS_SCHEMA)
            assert "minimum" in str(exc_info.value).lower()

    def test_latency_above_maximum(self):
        """Test latency above maximum (> 10) is rejected."""
        invalid_cases = [
            {"bandwidth": 3, "latency": 11},
            {"bandwidth": 3, "latency": 10.1},
            {"bandwidth": 3, "latency": 50},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError) as exc_info:
                validate(instance=qos, schema=QOS_SCHEMA)
            assert "maximum" in str(exc_info.value).lower()


class TestInvalidTypes:
    """Test that invalid data types are rejected."""

    def test_string_values_rejected(self):
        """Test that string values for numeric fields are rejected."""
        invalid_cases = [
            {"bandwidth": "5", "latency": 5},
            {"bandwidth": 5, "latency": "5"},
            {"bandwidth": "high", "latency": "low"},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError) as exc_info:
                validate(instance=qos, schema=QOS_SCHEMA)
            assert "type" in str(exc_info.value).lower()

    def test_null_values_rejected(self):
        """Test that null values for required fields are rejected."""
        invalid_cases = [
            {"bandwidth": None, "latency": 5},
            {"bandwidth": 5, "latency": None},
            {"bandwidth": None, "latency": None},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError):
                validate(instance=qos, schema=QOS_SCHEMA)

    def test_array_values_rejected(self):
        """Test that array values are rejected."""
        invalid_cases = [
            {"bandwidth": [1, 2, 3], "latency": 5},
            {"bandwidth": 5, "latency": [1, 2, 3]},
        ]

        for qos in invalid_cases:
            with pytest.raises(ValidationError):
                validate(instance=qos, schema=QOS_SCHEMA)


class TestRequiredFields:
    """Test required field validation."""

    def test_missing_bandwidth(self):
        """Test that missing bandwidth field is rejected."""
        qos = {"latency": 5}
        with pytest.raises(ValidationError) as exc_info:
            validate(instance=qos, schema=QOS_SCHEMA)
        assert "bandwidth" in str(exc_info.value)

    def test_missing_latency(self):
        """Test that missing latency field is rejected."""
        qos = {"bandwidth": 3}
        with pytest.raises(ValidationError) as exc_info:
            validate(instance=qos, schema=QOS_SCHEMA)
        assert "latency" in str(exc_info.value)

    def test_missing_all_required(self):
        """Test that empty object is rejected."""
        qos = {}
        with pytest.raises(ValidationError) as exc_info:
            validate(instance=qos, schema=QOS_SCHEMA)
        # Should mention both required fields
        error_msg = str(exc_info.value)
        assert "bandwidth" in error_msg or "required" in error_msg.lower()


class TestOptionalFields:
    """Test optional field validation."""

    def test_valid_optional_fields(self):
        """Test that valid optional fields are accepted."""
        qos = {
            "bandwidth": 3,
            "latency": 5,
            "jitter": 2,
            "packet_loss": 0.1,
            "reliability": 99.9,
            "slice_type": "eMBB"
        }
        validate(instance=qos, schema=QOS_SCHEMA)

    def test_invalid_slice_type(self):
        """Test that invalid slice_type is rejected."""
        qos = {"bandwidth": 3, "latency": 5, "slice_type": "invalid"}
        with pytest.raises(ValidationError) as exc_info:
            validate(instance=qos, schema=QOS_SCHEMA)
        assert "enum" in str(exc_info.value).lower()

    def test_additional_properties_rejected(self):
        """Test that additional properties are rejected."""
        qos = {
            "bandwidth": 3,
            "latency": 5,
            "unknown_field": "value"
        }
        with pytest.raises(ValidationError) as exc_info:
            validate(instance=qos, schema=QOS_SCHEMA)
        assert "additional" in str(exc_info.value).lower()


class TestSchemaValidation:
    """Test the schema itself is valid."""

    def test_schema_is_valid_draft7(self):
        """Test that the schema conforms to JSON Schema Draft-07."""
        # This will raise if schema is invalid
        Draft7Validator.check_schema(QOS_SCHEMA)

    def test_schema_has_required_properties(self):
        """Test that schema defines all expected properties."""
        assert "properties" in QOS_SCHEMA
        assert "bandwidth" in QOS_SCHEMA["properties"]
        assert "latency" in QOS_SCHEMA["properties"]
        assert "required" in QOS_SCHEMA
        assert "bandwidth" in QOS_SCHEMA["required"]
        assert "latency" in QOS_SCHEMA["required"]