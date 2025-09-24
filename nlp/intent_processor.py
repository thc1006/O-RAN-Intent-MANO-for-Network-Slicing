#!/usr/bin/env python3
"""
NLP Intent Processor for O-RAN Intent-based MANO
Translates natural language intents into QoS parameters for network slicing
"""

import json
import re
from dataclasses import asdict, dataclass
from enum import Enum
from typing import Dict, Tuple


class ServiceType(Enum):
    """Network slice service types"""

    EMBB = "enhanced_mobile_broadband"  # High bandwidth
    URLLC = "ultra_reliable_low_latency"  # Low latency
    MMTC = "massive_machine_type_communication"  # IoT
    GAMING = "gaming"  # Low latency gaming
    VIDEO = "video_streaming"  # High bandwidth video
    VOICE = "voice_over_ip"  # Voice services
    IOT = "internet_of_things"  # IoT services
    CRITICAL = "mission_critical"  # Critical services


@dataclass
class QoSParameters:
    """QoS parameters for network slice"""

    max_latency_ms: float
    min_throughput_mbps: float
    max_packet_loss_rate: float
    max_jitter_ms: float
    reliability_percent: float
    priority: int  # 1-10, higher is more important

    def to_json(self) -> str:
        """Convert to JSON string"""
        return json.dumps(asdict(self), indent=2)


@dataclass
class IntentResult:
    """Result of intent processing"""

    original_intent: str
    service_type: ServiceType
    qos_parameters: QoSParameters
    placement_hints: Dict[str, str]
    confidence: float  # 0-1 confidence score

    def to_json(self) -> str:
        """Convert to JSON string"""
        data = {
            "original_intent": self.original_intent,
            "service_type": self.service_type.value,
            "qos_parameters": asdict(self.qos_parameters),
            "placement_hints": self.placement_hints,
            "confidence": self.confidence,
        }
        return json.dumps(data, indent=2)


class IntentProcessor:
    """Natural language intent processor"""

    def __init__(self):
        """Initialize intent processor with patterns and mappings"""
        # Keywords for service type detection
        self.service_keywords = {
            ServiceType.GAMING: [
                "gaming",
                "game",
                "ar/vr",
                "vr",
                "ar",
                "augmented",
                "virtual",
                "interactive",
                "real-time gaming",
                "cloud gaming",
            ],
            ServiceType.VIDEO: [
                "video",
                "streaming",
                "4k",
                "8k",
                "hd",
                "ultra-hd",
                "broadcast",
                "live stream",
                "media",
                "content delivery",
            ],
            ServiceType.URLLC: [
                "ultra-low latency",
                "low latency",
                "critical",
                "real-time",
                "autonomous",
                "vehicle",
                "v2x",
                "industrial",
                "automation",
                "robotics",
                "surgery",
                "telemedicine",
            ],
            ServiceType.EMBB: [
                "high bandwidth",
                "broadband",
                "high throughput",
                "capacity",
                "data intensive",
                "mobile broadband",
                "high speed",
            ],
            ServiceType.VOICE: [
                "voice",
                "voip",
                "call",
                "telephony",
                "audio",
                "conference",
            ],
            ServiceType.IOT: [
                "iot",
                "sensor",
                "monitoring",
                "smart city",
                "smart home",
                "telemetry",
                "metering",
                "tracking",
            ],
            ServiceType.CRITICAL: [
                "mission critical",
                "emergency",
                "public safety",
                "first responder",
                "critical communication",
            ],
            ServiceType.MMTC: [
                "massive iot",
                "massive connectivity",
                "machine type",
                "m2m",
                "sensor network",
            ],
        }

        # QoS requirement patterns
        self.qos_patterns = {
            "latency": [
                (r"(\d+)\s*ms\s*latency", lambda x: float(x)),
                (r"latency\s*[<≤]\s*(\d+)\s*ms", lambda x: float(x)),
                (r"ultra[-\s]?low\s*latency", lambda: 10.0),
                (r"low\s*latency", lambda: 20.0),
                (r"moderate\s*latency", lambda: 50.0),
            ],
            "throughput": [
                (r"(\d+\.?\d*)\s*[Mm]bps", lambda x: float(x)),
                (r"(\d+\.?\d*)\s*[Gg]bps", lambda x: float(x) * 1000),
                (r"high\s*bandwidth", lambda: 100.0),
                (r"moderate\s*bandwidth", lambda: 10.0),
                (r"low\s*bandwidth", lambda: 1.0),
            ],
            "reliability": [
                (r"(\d+\.?\d*)\s*%\s*reliability", lambda x: float(x)),
                (r"(\d+)\s*nines", lambda x: 100 - 10 ** (-int(x)) * 100),
                (r"ultra[-\s]?reliable", lambda: 99.999),
                (r"high\s*reliability", lambda: 99.99),
                (r"moderate\s*reliability", lambda: 99.9),
            ],
        }

        # Default QoS profiles for service types
        self.default_qos = {
            ServiceType.GAMING: QoSParameters(
                max_latency_ms=10.0,
                min_throughput_mbps=5.0,
                max_packet_loss_rate=0.001,
                max_jitter_ms=2.0,
                reliability_percent=99.9,
                priority=8,
            ),
            ServiceType.VIDEO: QoSParameters(
                max_latency_ms=100.0,
                min_throughput_mbps=25.0,
                max_packet_loss_rate=0.01,
                max_jitter_ms=10.0,
                reliability_percent=99.0,
                priority=6,
            ),
            ServiceType.URLLC: QoSParameters(
                max_latency_ms=5.0,
                min_throughput_mbps=1.0,
                max_packet_loss_rate=0.00001,
                max_jitter_ms=1.0,
                reliability_percent=99.999,
                priority=10,
            ),
            ServiceType.EMBB: QoSParameters(
                max_latency_ms=50.0,
                min_throughput_mbps=100.0,
                max_packet_loss_rate=0.001,
                max_jitter_ms=5.0,
                reliability_percent=99.9,
                priority=7,
            ),
            ServiceType.VOICE: QoSParameters(
                max_latency_ms=20.0,
                min_throughput_mbps=0.1,
                max_packet_loss_rate=0.001,
                max_jitter_ms=3.0,
                reliability_percent=99.99,
                priority=8,
            ),
            ServiceType.IOT: QoSParameters(
                max_latency_ms=1000.0,
                min_throughput_mbps=0.01,
                max_packet_loss_rate=0.01,
                max_jitter_ms=100.0,
                reliability_percent=99.0,
                priority=3,
            ),
            ServiceType.CRITICAL: QoSParameters(
                max_latency_ms=10.0,
                min_throughput_mbps=1.0,
                max_packet_loss_rate=0.00001,
                max_jitter_ms=1.0,
                reliability_percent=99.999,
                priority=10,
            ),
            ServiceType.MMTC: QoSParameters(
                max_latency_ms=10000.0,
                min_throughput_mbps=0.001,
                max_packet_loss_rate=0.1,
                max_jitter_ms=1000.0,
                reliability_percent=95.0,
                priority=2,
            ),
        }

    def process_intent(self, intent: str) -> IntentResult:
        """
        Process natural language intent and extract QoS parameters

        Args:
            intent: Natural language description of network slice requirements

        Returns:
            IntentResult with service type, QoS parameters, and confidence
        """
        intent_lower = intent.lower()

        # Detect service type
        service_type, type_confidence = self._detect_service_type(intent_lower)

        # Get base QoS parameters for service type
        qos_params = self._get_base_qos(service_type)

        # Extract and override specific QoS requirements from intent
        extracted_qos, qos_confidence = self._extract_qos_requirements(intent_lower)
        qos_params = self._merge_qos_parameters(qos_params, extracted_qos)

        # Extract placement hints
        placement_hints = self._extract_placement_hints(intent_lower)

        # Calculate overall confidence
        confidence = (type_confidence + qos_confidence) / 2

        return IntentResult(
            original_intent=intent,
            service_type=service_type,
            qos_parameters=qos_params,
            placement_hints=placement_hints,
            confidence=confidence,
        )

    def _detect_service_type(self, intent: str) -> Tuple[ServiceType, float]:
        """Detect service type from intent text"""
        scores = {}

        for service_type, keywords in self.service_keywords.items():
            score = 0
            for keyword in keywords:
                if keyword in intent:
                    score += 1
            scores[service_type] = score

        # Get service type with highest score
        if not scores or max(scores.values()) == 0:
            # Default to EMBB if no specific type detected
            return ServiceType.EMBB, 0.5

        best_type = max(scores, key=scores.get)
        max_score = scores[best_type]
        confidence = min(max_score / 3.0, 1.0)  # Normalize confidence

        return best_type, confidence

    def _extract_qos_requirements(self, intent: str) -> Tuple[Dict, float]:
        """Extract specific QoS requirements from intent"""
        extracted = {}
        matches_found = 0

        # Extract latency requirements
        for pattern, extractor in self.qos_patterns["latency"]:
            match = re.search(pattern, intent)
            if match:
                if match.groups():
                    extracted["max_latency_ms"] = extractor(match.group(1))
                else:
                    extracted["max_latency_ms"] = extractor()
                matches_found += 1
                break

        # Extract throughput requirements
        for pattern, extractor in self.qos_patterns["throughput"]:
            match = re.search(pattern, intent)
            if match:
                if match.groups():
                    extracted["min_throughput_mbps"] = extractor(match.group(1))
                else:
                    extracted["min_throughput_mbps"] = extractor()
                matches_found += 1
                break

        # Extract reliability requirements
        for pattern, extractor in self.qos_patterns["reliability"]:
            match = re.search(pattern, intent)
            if match:
                if match.groups():
                    extracted["reliability_percent"] = extractor(match.group(1))
                else:
                    extracted["reliability_percent"] = extractor()
                matches_found += 1
                break

        # Extract packet loss requirements
        if "no packet loss" in intent or "zero packet loss" in intent:
            extracted["max_packet_loss_rate"] = 0.00001
            matches_found += 1
        elif "low packet loss" in intent:
            extracted["max_packet_loss_rate"] = 0.001
            matches_found += 1

        # Extract jitter requirements
        if "low jitter" in intent or "stable" in intent:
            extracted["max_jitter_ms"] = 2.0
            matches_found += 1
        elif "no jitter" in intent:
            extracted["max_jitter_ms"] = 0.5
            matches_found += 1

        confidence = min(matches_found / 3.0, 1.0)  # Normalize confidence
        return extracted, confidence

    def _get_base_qos(self, service_type: ServiceType) -> QoSParameters:
        """Get base QoS parameters for service type"""
        return QoSParameters(**asdict(self.default_qos[service_type]))

    def _merge_qos_parameters(
        self, base: QoSParameters, extracted: Dict
    ) -> QoSParameters:
        """Merge extracted QoS parameters with base parameters"""
        params = asdict(base)
        params.update(extracted)
        return QoSParameters(**params)

    def _extract_placement_hints(self, intent: str) -> Dict[str, str]:
        """Extract placement hints from intent"""
        hints = {}

        # Edge placement
        if any(
            word in intent for word in ["edge", "near user", "close to user", "local"]
        ):
            hints["cloud_type"] = "edge"
        elif any(word in intent for word in ["regional", "metro", "city"]):
            hints["cloud_type"] = "regional"
        elif any(word in intent for word in ["central", "core", "datacenter"]):
            hints["cloud_type"] = "central"

        # Geographic hints
        location_patterns = [
            (r"in\s+(\w+)\s+region", "region"),
            (r"at\s+(\w+)\s+site", "site"),
            (r"near\s+(\w+)", "near"),
        ]

        for pattern, hint_type in location_patterns:
            match = re.search(pattern, intent)
            if match:
                hints[hint_type] = match.group(1)

        # Affinity hints
        if "isolated" in intent or "dedicated" in intent:
            hints["affinity"] = "anti-affinity"
        elif "colocated" in intent or "together" in intent:
            hints["affinity"] = "affinity"

        return hints


def main():
    """Example usage of IntentProcessor"""
    processor = IntentProcessor()

    # Example intents
    test_intents = [
        "Create a low-latency network slice for AR/VR gaming with guaranteed 10ms latency",
        "I need a high bandwidth slice for 4K video streaming with at least 25 Mbps",
        "Deploy an ultra-reliable slice for autonomous vehicle communication with 5 nines reliability",
        "Set up IoT monitoring with low bandwidth requirements at the edge",
        "Mission critical communication for emergency services with ultra-low latency",
        "Gaming service requiring less than 6.3ms latency and 1 Mbps throughput",
        "High bandwidth video streaming tolerating up to 20ms latency with 4.57 Mbps",
    ]

    for intent in test_intents:
        print(f"\nIntent: {intent}")
        print("-" * 80)
        result = processor.process_intent(intent)
        print(f"Service Type: {result.service_type.value}")
        print(f"Confidence: {result.confidence:.2f}")
        print("QoS Parameters:")
        print(f"  Max Latency: {result.qos_parameters.max_latency_ms} ms")
        print(f"  Min Throughput: {result.qos_parameters.min_throughput_mbps} Mbps")
        print(f"  Max Packet Loss: {result.qos_parameters.max_packet_loss_rate}")
        print(f"  Max Jitter: {result.qos_parameters.max_jitter_ms} ms")
        print(f"  Reliability: {result.qos_parameters.reliability_percent}%")
        print(f"  Priority: {result.qos_parameters.priority}")
        if result.placement_hints:
            print(f"Placement Hints: {result.placement_hints}")


if __name__ == "__main__":
    main()
