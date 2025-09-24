#!/usr/bin/env python3
"""
Schema Validator for O-RAN Intent-Based MANO
Validates QoS JSON against O-RAN schema specifications
"""

import json
from pathlib import Path
from typing import Any, Dict, List, Optional

import jsonschema


class SchemaValidator:
    """Validates QoS specifications against O-RAN schema"""

    def __init__(self, schema_path: Optional[str] = None):
        """
        Initialize the schema validator

        Args:
            schema_path: Path to the JSON schema file
        """
        if schema_path is None:
            # Use default schema in the same directory
            self.schema_path = Path(__file__).parent / "schema.json"
        else:
            self.schema_path = Path(schema_path)
        self.schema = self._load_schema()

    def _load_schema(self) -> Dict[str, Any]:
        """
        Load the JSON schema from file

        Returns:
            Parsed JSON schema

        Raises:
            FileNotFoundError: If schema file doesn't exist
            json.JSONDecodeError: If schema is invalid JSON
        """
        if not self.schema_path.exists():
            raise FileNotFoundError(f"Schema file not found: {self.schema_path}")

        with open(self.schema_path, "r", encoding="utf-8") as f:
            return json.load(f)

    def validate(self, data: Dict[str, Any]) -> bool:
        """
        Validate data against the schema

        Args:
            data: Data to validate

        Returns:
            True if validation passes

        Raises:
            jsonschema.ValidationError: If validation fails
        """
        jsonschema.validate(instance=data, schema=self.schema)
        return True

    def validate_safe(self, data: Dict[str, Any]) -> tuple[bool, Optional[str]]:
        """
        Validate data against the schema without raising exceptions

        Args:
            data: Data to validate

        Returns:
            Tuple of (is_valid, error_message)
        """
        try:
            self.validate(data)
            return True, None
        except jsonschema.ValidationError as e:
            return False, str(e)
        except (TypeError, ValueError) as e:
            return False, f"Unexpected error: {str(e)}"

    def get_schema_summary(self) -> Dict[str, Any]:
        """
        Get a summary of the schema structure

        Returns:
            Dictionary with schema summary information
        """
        return {
            "type": self.schema.get("type", "unknown"),
            "properties": list(self.schema.get("properties", {}).keys()),
            "required": self.schema.get("required", []),
            "title": self.schema.get("title", "O-RAN QoS Schema"),
            "description": self.schema.get("description", ""),
        }

    def validate_slice_type(self, slice_type: str) -> bool:
        """
        Validate if slice type is allowed

        Args:
            slice_type: Slice type to validate

        Returns:
            True if slice type is valid
        """
        valid_types = ["eMBB", "URLLC", "mMTC"]
        return slice_type in valid_types

    def validate_qos_parameters(self, qos: Dict[str, Any]) -> tuple[bool, List[str]]:
        """
        Validate QoS parameters against thesis requirements

        Args:
            qos: QoS parameters dictionary

        Returns:
            Tuple of (is_valid, list_of_issues)
        """
        issues = []

        # Check required fields
        required_fields = ["throughputMbps", "latencyMs", "packetLossRate"]
        for field in required_fields:
            if field not in qos:
                issues.append(f"Missing required field: {field}")

        # Validate throughput
        if "throughputMbps" in qos:
            throughput = qos["throughputMbps"]
            if not isinstance(throughput, (int, float)):
                issues.append("throughputMbps must be a number")
            elif throughput <= 0:
                issues.append("throughputMbps must be positive")
            elif throughput > 1000:
                issues.append("throughputMbps exceeds maximum (1000 Mbps)")

        # Validate latency
        if "latencyMs" in qos:
            latency = qos["latencyMs"]
            if not isinstance(latency, (int, float)):
                issues.append("latencyMs must be a number")
            elif latency <= 0:
                issues.append("latencyMs must be positive")
            elif latency < 1:
                issues.append("Sub-millisecond latency not supported")

        # Validate packet loss
        if "packetLossRate" in qos:
            loss = qos["packetLossRate"]
            if not isinstance(loss, (int, float)):
                issues.append("packetLossRate must be a number")
            elif not 0 <= loss <= 1:
                issues.append("packetLossRate must be between 0 and 1")

        # Validate priority if present
        if "priority" in qos:
            priority = qos["priority"]
            if not isinstance(priority, int):
                issues.append("priority must be an integer")
            elif not 1 <= priority <= 10:
                issues.append("priority must be between 1 and 10")

        # Validate optional fields
        if "jitterMs" in qos:
            jitter = qos["jitterMs"]
            if not isinstance(jitter, (int, float)):
                issues.append("jitterMs must be a number")
            elif jitter < 0:
                issues.append("jitterMs cannot be negative")

        if "reliability" in qos:
            reliability = qos["reliability"]
            if not isinstance(reliability, (int, float)):
                issues.append("reliability must be a number")
            elif not 0 <= reliability <= 1:
                issues.append("reliability must be between 0 and 1")

        return len(issues) == 0, issues
