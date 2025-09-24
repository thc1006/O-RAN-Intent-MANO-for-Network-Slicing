#!/usr/bin/env python3
"""
High-Performance Intent Processing Cache
Provides aggressive caching and pre-computation for intent processing
"""

import concurrent.futures
import hashlib
import threading
import time
from collections import OrderedDict
from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Tuple

import regex as re  # More efficient than standard re

from .intent_processor import IntentProcessor, IntentResult, ServiceType


@dataclass
class CacheEntry:
    """Cached intent processing result"""

    result: IntentResult
    timestamp: float
    hit_count: int
    processing_time: float

    def is_expired(self, ttl: float) -> bool:
        """Check if cache entry has expired"""
        return time.time() - self.timestamp > ttl


class PerformanceIntentCache:
    """High-performance intent processing cache with pre-computation"""

    def __init__(
        self, max_size: int = 10000, ttl: float = 3600, precompute_common: bool = True
    ):
        """
        Initialize performance cache

        Args:
            max_size: Maximum cache entries (LRU eviction)
            ttl: Time-to-live for cache entries in seconds
            precompute_common: Whether to pre-compute common intent patterns
        """
        self.max_size = max_size
        self.ttl = ttl
        self.cache: OrderedDict[str, CacheEntry] = OrderedDict()
        self.lock = threading.RLock()
        self.processor = IntentProcessor()

        # Performance counters
        self.stats = {
            "hits": 0,
            "misses": 0,
            "evictions": 0,
            "precomputed": 0,
            "total_processing_time": 0.0,
        }

        # Pre-compiled regex patterns for performance
        self._compile_patterns()

        # Pre-compute common intents if enabled
        if precompute_common:
            self._precompute_common_intents()

    def _compile_patterns(self):
        """Pre-compile regex patterns for better performance"""
        self.compiled_patterns = {}

        # Latency patterns
        self.compiled_patterns["latency"] = [
            (
                re.compile(r"(\d+(?:\.\d+)?)\s*ms\s*latency", re.IGNORECASE),
                float,
            ),
            (
                re.compile(r"latency\s*[<≤]\s*(\d+(?:\.\d+)?)\s*ms", re.IGNORECASE),
                float,
            ),
            (re.compile(r"ultra[-\s]?low\s*latency", re.IGNORECASE), 10.0),
            (re.compile(r"low\s*latency", re.IGNORECASE), 20.0),
            (re.compile(r"moderate\s*latency", re.IGNORECASE), 50.0),
        ]

        # Throughput patterns
        self.compiled_patterns["throughput"] = [
            (
                re.compile(r"(\d+(?:\.\d+)?)\s*[Mm]bps", re.IGNORECASE),
                float,
            ),
            (
                re.compile(r"(\d+(?:\.\d+)?)\s*[Gg]bps", re.IGNORECASE),
                lambda x: float(x) * 1000,
            ),
            (re.compile(r"high\s*bandwidth", re.IGNORECASE), lambda: 100.0),
            (re.compile(r"moderate\s*bandwidth", re.IGNORECASE), lambda: 10.0),
            (re.compile(r"low\s*bandwidth", re.IGNORECASE), lambda: 1.0),
        ]

        # Service type patterns
        self.compiled_patterns["service_types"] = {}
        for service_type, keywords in self.processor.service_keywords.items():
            pattern = "|".join(re.escape(keyword) for keyword in keywords)
            self.compiled_patterns["service_types"][service_type] = re.compile(
                rf"\b(?:{pattern})\b", re.IGNORECASE
            )

    def _precompute_common_intents(self):
        """Pre-compute common intent patterns"""
        common_intents = [
            # Gaming scenarios
            "Create a low-latency network slice for AR/VR gaming with guaranteed 10ms latency",
            "Gaming service requiring less than 6.3ms latency and 1 Mbps throughput",
            "Ultra-low latency gaming slice with 5ms latency",
            # Video streaming
            "I need a high bandwidth slice for 4K video streaming with at least 25 Mbps",
            "High bandwidth video streaming tolerating up to 20ms latency with 4.57 Mbps",
            "Video streaming service with 10 Mbps bandwidth",
            # URLLC scenarios
            "Deploy ultra-reliable slice for autonomous vehicles with 5 nines reliability",
            "Ultra-low latency slice for critical applications with 1ms latency",
            "Mission critical communication for emergency services",
            # IoT scenarios
            "Set up IoT monitoring with low bandwidth requirements at the edge",
            "Massive IoT connectivity for smart city sensors",
            "IoT slice with low power requirements",
            # eMBB scenarios
            "High bandwidth mobile broadband with 100 Mbps",
            "Enhanced mobile broadband for dense urban areas",
            "Mobile broadband slice with high capacity",
            # Thesis-specific scenarios
            "High bandwidth video streaming tolerating up to 20ms latency with 4.57 Mbps",
            "Gaming service requiring less than 6.3ms latency and 0.93 Mbps throughput",
            "IoT monitoring with 2.77 Mbps bandwidth and 15.7ms latency tolerance",
        ]

        print("Pre-computing common intent patterns...")
        start_time = time.time()

        # Use thread pool for parallel pre-computation
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
            futures = [
                executor.submit(self._compute_and_cache, intent, precompute=True)
                for intent in common_intents
            ]

            for future in concurrent.futures.as_completed(futures):
                try:
                    future.result()
                    self.stats["precomputed"] += 1
                except (ValueError, KeyError, TypeError) as e:
                    print(f"Error pre-computing intent: {e}")

        precompute_time = time.time() - start_time
        print(
            f"Pre-computed {self.stats['precomputed']} intent patterns in {precompute_time:.2f}s"
        )

    def _compute_hash(self, intent: str) -> str:
        """Compute hash for intent caching"""
        # Normalize intent for consistent hashing
        normalized = intent.lower().strip()
        # Remove extra whitespace
        normalized = re.sub(r"\s+", " ", normalized)
        return hashlib.sha256(normalized.encode()).hexdigest()

    def _fast_service_type_detection(self, intent: str) -> Tuple[ServiceType, float]:
        """Fast service type detection using pre-compiled patterns"""
        intent_lower = intent.lower()
        scores = {}

        for service_type, pattern in self.compiled_patterns["service_types"].items():
            matches = pattern.findall(intent_lower)
            scores[service_type] = len(matches)

        if not scores or max(scores.values()) == 0:
            return ServiceType.EMBB, 0.5

        best_type = max(scores, key=lambda k: scores[k])
        max_score = scores[best_type]
        confidence = min(max_score / 3.0, 1.0)

        return best_type, confidence

    def _fast_qos_extraction(self, intent: str) -> Tuple[Dict, float]:
        """Fast QoS parameter extraction using pre-compiled patterns"""
        extracted = {}
        matches_found = 0

        # Extract latency
        for pattern, extractor in self.compiled_patterns["latency"]:
            match = pattern.search(intent)
            if match:
                if match.groups():
                    extracted["max_latency_ms"] = extractor(match.group(1))
                else:
                    extracted["max_latency_ms"] = extractor()
                matches_found += 1
                break

        # Extract throughput
        for pattern, extractor in self.compiled_patterns["throughput"]:
            match = pattern.search(intent)
            if match:
                if match.groups():
                    extracted["min_throughput_mbps"] = extractor(match.group(1))
                else:
                    extracted["min_throughput_mbps"] = extractor()
                matches_found += 1
                break

        # Quick reliability checks
        if "ultra-reliable" in intent.lower() or "5 nines" in intent.lower():
            extracted["reliability_percent"] = 99.999
            matches_found += 1
        elif "high reliability" in intent.lower():
            extracted["reliability_percent"] = 99.99
            matches_found += 1

        confidence = min(matches_found / 3.0, 1.0)
        return extracted, confidence

    def _compute_and_cache(self, intent: str, precompute: bool = False) -> IntentResult:
        """Compute intent result and cache it"""
        start_time = time.time()

        # Use optimized processing for better performance
        service_type, type_confidence = self._fast_service_type_detection(intent)

        # Get base QoS parameters
        qos_params = self.processor.get_base_qos(service_type)

        # Extract specific QoS requirements
        extracted_qos, qos_confidence = self._fast_qos_extraction(intent)
        qos_params = self.processor.merge_qos_parameters(qos_params, extracted_qos)

        # Extract placement hints (use original method for now)
        placement_hints = self.processor.extract_placement_hints(intent.lower())

        # Calculate confidence
        confidence = (type_confidence + qos_confidence) / 2

        result = IntentResult(
            original_intent=intent,
            service_type=service_type,
            qos_parameters=qos_params,
            placement_hints=placement_hints,
            confidence=confidence,
        )

        processing_time = time.time() - start_time

        # Cache the result
        if not precompute:  # Only cache during normal operation
            cache_key = self._compute_hash(intent)
            with self.lock:
                self.cache[cache_key] = CacheEntry(
                    result=result,
                    timestamp=time.time(),
                    hit_count=0,
                    processing_time=processing_time,
                )
                self._evict_if_needed()

        self.stats["total_processing_time"] += processing_time
        return result

    def _evict_if_needed(self):
        """Evict oldest entries if cache is full"""
        while len(self.cache) > self.max_size:
            self.cache.popitem(last=False)
            self.stats["evictions"] += 1

    def process_intent(self, intent: str) -> IntentResult:
        """
        Process intent with caching

        Args:
            intent: Natural language intent

        Returns:
            IntentResult with optimized processing
        """
        cache_key = self._compute_hash(intent)

        with self.lock:
            # Check cache first
            if cache_key in self.cache:
                entry = self.cache[cache_key]

                # Check if expired
                if not entry.is_expired(self.ttl):
                    # Move to end (LRU)
                    self.cache.move_to_end(cache_key)
                    entry.hit_count += 1
                    self.stats["hits"] += 1
                    return entry.result
                # Remove expired entry
                del self.cache[cache_key]

            # Cache miss - compute result
            self.stats["misses"] += 1

        # Compute outside of lock to allow concurrent access
        return self._compute_and_cache(intent)

    def batch_process(
        self, intents: List[str], max_workers: int = 4
    ) -> List[IntentResult]:
        """
        Process multiple intents in parallel

        Args:
            intents: List of intent strings
            max_workers: Maximum number of worker threads

        Returns:
            List of IntentResult objects
        """
        results: List[Optional[IntentResult]] = [None] * len(intents)

        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            # Submit all tasks
            future_to_index = {
                executor.submit(self.process_intent, intent): i
                for i, intent in enumerate(intents)
            }

            # Collect results in order
            for future in concurrent.futures.as_completed(future_to_index):
                index = future_to_index[future]
                try:
                    results[index] = future.result()
                except (ValueError, RuntimeError) as e:
                    print(f"Error processing intent {index}: {e}")
                    # Create error result
                    results[index] = IntentResult(
                        original_intent=intents[index],
                        service_type=ServiceType.EMBB,
                        qos_parameters=self.processor.default_qos[ServiceType.EMBB],
                        placement_hints={},
                        confidence=0.0,
                    )

        return [r for r in results if r is not None]

    def warm_cache(self, intents: List[str]):
        """Warm cache with provided intents"""
        print(f"Warming cache with {len(intents)} intents...")
        start_time = time.time()

        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
            futures = [
                executor.submit(self.process_intent, intent) for intent in intents
            ]
            for future in concurrent.futures.as_completed(futures):
                try:
                    future.result()
                except (ValueError, RuntimeError) as e:
                    print(f"Error warming cache: {e}")

        warm_time = time.time() - start_time
        print(f"Cache warmed in {warm_time:.2f}s")

    def get_statistics(self) -> Dict[str, Any]:
        """Get cache performance statistics"""
        with self.lock:
            total_requests = self.stats["hits"] + self.stats["misses"]
            hit_rate = self.stats["hits"] / total_requests if total_requests > 0 else 0

            avg_processing_time = (
                self.stats["total_processing_time"] / self.stats["misses"]
                if self.stats["misses"] > 0
                else 0
            )

            return {
                "cache_size": len(self.cache),
                "max_size": self.max_size,
                "hit_rate": hit_rate,
                "total_requests": total_requests,
                "hits": self.stats["hits"],
                "misses": self.stats["misses"],
                "evictions": self.stats["evictions"],
                "precomputed": self.stats["precomputed"],
                "avg_processing_time_ms": avg_processing_time * 1000,
            }

    def clear_cache(self):
        """Clear all cache entries"""
        with self.lock:
            self.cache.clear()
            # Reset stats except precomputed
            precomputed = self.stats["precomputed"]
            self.stats = {
                "hits": 0,
                "misses": 0,
                "evictions": 0,
                "precomputed": precomputed,
                "total_processing_time": 0.0,
            }


# Global cache instance for reuse
_global_cache: Optional[PerformanceIntentCache] = None


def get_cached_processor() -> PerformanceIntentCache:
    """Get or create global cached processor instance"""
    global _global_cache  # pylint: disable=global-statement
    if _global_cache is None:
        _global_cache = PerformanceIntentCache()
    return _global_cache


def benchmark_performance():
    """Benchmark performance improvements"""
    print("Benchmarking intent processing performance...")

    # Test intents from thesis
    test_intents = [
        "High bandwidth video streaming tolerating up to 20ms latency with 4.57 Mbps",
        "Gaming service requiring less than 6.3ms latency and 0.93 Mbps throughput",
        "IoT monitoring with 2.77 Mbps bandwidth and 15.7ms latency tolerance",
        "Create a low-latency network slice for AR/VR gaming with guaranteed 10ms latency",
        "Deploy an ultra-reliable slice for autonomous vehicle communication",
        "Mission critical communication for emergency services",
    ]

    # Benchmark original processor
    original_processor = IntentProcessor()
    original_times = []

    for intent in test_intents:
        start_time = time.time()
        original_processor.process_intent(intent)
        original_times.append(time.time() - start_time)

    original_avg = sum(original_times) / len(original_times)

    # Benchmark cached processor
    cached_processor = PerformanceIntentCache(precompute_common=True)

    # First run (cache misses)
    cached_times_miss = []
    for intent in test_intents:
        start_time = time.time()
        cached_processor.process_intent(intent)
        cached_times_miss.append(time.time() - start_time)

    # Second run (cache hits)
    cached_times_hit = []
    for intent in test_intents:
        start_time = time.time()
        cached_processor.process_intent(intent)
        cached_times_hit.append(time.time() - start_time)

    cached_miss_avg = sum(cached_times_miss) / len(cached_times_miss)
    cached_hit_avg = sum(cached_times_hit) / len(cached_times_hit)

    print("\n=== Performance Benchmark Results ===")
    print(f"Original processor average: {original_avg*1000:.2f}ms")
    print(f"Cached processor (miss) average: {cached_miss_avg*1000:.2f}ms")
    print(f"Cached processor (hit) average: {cached_hit_avg*1000:.2f}ms")
    print(f"Cache miss speedup: {original_avg/cached_miss_avg:.2f}x")
    print(f"Cache hit speedup: {original_avg/cached_hit_avg:.2f}x")
    print(f"Cache statistics: {cached_processor.get_statistics()}")


if __name__ == "__main__":
    benchmark_performance()
