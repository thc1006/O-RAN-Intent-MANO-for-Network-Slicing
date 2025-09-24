#!/usr/bin/env python3
"""
E2E Test Harness for O-RAN Intent-Based MANO System
Validates deployment automation and metrics collection
"""

import argparse
import json
import logging
import subprocess
import sys
import time
from pathlib import Path
from typing import Dict, Optional

import yaml


class TestHarness:
    """Main test harness for E2E deployment validation"""

    def __init__(self, config_dir: str = "config", results_dir: str = "results"):
        self.config_dir = Path(config_dir)
        self.results_dir = Path(results_dir)
        self.logger = self._setup_logging()
        self.thresholds = self._load_thresholds()

    def _setup_logging(self) -> logging.Logger:
        """Setup logging configuration"""
        logging.basicConfig(
            level=logging.INFO,
            format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        )
        return logging.getLogger(__name__)

    def _load_thresholds(self) -> Dict:
        """Load validation thresholds"""
        thresholds_file = self.config_dir / "thresholds.yaml"
        if not thresholds_file.exists():
            self.logger.warning(f"Thresholds file not found: {thresholds_file}")
            return {}

        with open(thresholds_file) as f:
            return yaml.safe_load(f)

    def run_prerequisite_checks(self) -> bool:
        """Run prerequisite checks before testing"""
        self.logger.info("Running prerequisite checks...")

        checks = [
            ("kubectl cluster-info", "Kubernetes cluster connectivity"),
            ("which yq", "yq command availability"),
            ("which bc", "bc command availability"),
            ("python3 -c 'import json, subprocess, time'", "Python dependencies"),
        ]

        for cmd, description in checks:
            if not self._run_command(cmd, description):
                return False

        # Check if metrics server is available
        if not self._run_command(
            "kubectl top nodes", "Metrics server availability", fail_ok=True
        ):
            self.logger.warning(
                "Metrics server not available - some metrics may be missing"
            )

        return True

    def run_deployment_test(self, series: str = "fast") -> Dict:
        """Run complete deployment test for a series"""
        self.logger.info(f"Running {series} series deployment test...")

        scenarios = ["embb", "urllc", "miot"]
        results = {
            "series": series,
            "scenarios": {},
            "validation": {"passed": True, "details": {}},
            "start_time": time.time(),
        }

        for scenario in scenarios:
            self.logger.info(f"Testing scenario: {scenario}")
            scenario_result = self._test_scenario(scenario, series)
            results["scenarios"][scenario] = scenario_result

            # Validate timing
            target = self._get_target_time(scenario, series)
            tolerance = self._get_tolerance(scenario, series)
            passed = self._validate_timing(
                scenario_result["duration"], target, tolerance
            )

            results["validation"]["details"][scenario] = {
                "actual": scenario_result["duration"],
                "target": target,
                "tolerance": tolerance,
                "passed": passed,
            }

            if not passed:
                results["validation"]["passed"] = False

        results["end_time"] = time.time()
        results["total_duration"] = results["end_time"] - results["start_time"]

        return results

    def _test_scenario(self, scenario: str, series: str) -> Dict:
        """Test a single deployment scenario"""
        start_time = time.time()

        # Run deployment script
        cmd = f"./run_suite.sh {series} {scenario}"
        success = self._run_command(cmd, f"{scenario} deployment")

        end_time = time.time()
        duration = end_time - start_time

        # Collect metrics
        metrics = self._collect_scenario_metrics(scenario)

        return {
            "scenario": scenario,
            "series": series,
            "duration": duration,
            "success": success,
            "metrics": metrics,
            "timestamp": time.time(),
        }

    def _collect_scenario_metrics(self, scenario: str) -> Dict:
        """Collect metrics for a scenario"""
        metrics = {"resource_usage": {}, "bottlenecks": {}, "performance": {}}

        try:
            # Collect resource metrics
            cmd = f"python3 collect_metrics.py collect_system --scenario {scenario} --output -"
            result = subprocess.run(
                cmd, shell=True, capture_output=True, text=True, timeout=30
            )

            if result.returncode == 0 and result.stdout:
                metrics.update(json.loads(result.stdout))

        except Exception as e:
            self.logger.warning(f"Failed to collect metrics for {scenario}: {e}")

        return metrics

    def _get_target_time(self, scenario: str, series: str) -> float:
        """Get target deployment time for scenario"""
        try:
            return self.thresholds["deployment_times"][f"{series}_series"][scenario][
                "target"
            ]
        except KeyError:
            # Default targets
            defaults = {
                "fast": {"embb": 407, "urllc": 353, "miot": 257},
                "slow": {"embb": 532, "urllc": 292, "miot": 220},
            }
            return defaults[series][scenario]

    def _get_tolerance(self, scenario: str, series: str) -> float:
        """Get tolerance for scenario timing"""
        try:
            return self.thresholds["deployment_times"][f"{series}_series"][scenario][
                "tolerance"
            ]
        except KeyError:
            return 20  # Default 20 second tolerance

    def _validate_timing(self, actual: float, target: float, tolerance: float) -> bool:
        """Validate if timing is within acceptable range"""
        return abs(actual - target) <= tolerance

    def run_performance_validation(self, results: Dict) -> Dict:
        """Run performance validation tests"""
        self.logger.info("Running performance validation...")

        validation = {
            "throughput_tests": {},
            "latency_tests": {},
            "resource_validation": {},
            "passed": True,
        }

        # Validate throughput with iPerf3
        for scenario in ["embb", "urllc", "miot"]:
            if scenario in results["scenarios"]:
                validation["throughput_tests"][scenario] = self._validate_throughput(
                    scenario
                )
                validation["latency_tests"][scenario] = self._validate_latency(scenario)

        # Validate resource usage
        validation["resource_validation"] = self._validate_resource_usage(results)

        # Overall validation
        for test_type in ["throughput_tests", "latency_tests", "resource_validation"]:
            if isinstance(validation[test_type], dict):
                for result in validation[test_type].values():
                    if isinstance(result, dict) and not result.get("passed", True):
                        validation["passed"] = False
                        break
            elif isinstance(validation[test_type], dict) and not validation[
                test_type
            ].get("passed", True):
                validation["passed"] = False

        return validation

    def _validate_throughput(self, scenario: str) -> Dict:
        """Validate throughput for scenario"""
        target_bandwidth = {"embb": 4.57, "urllc": 0.93, "miot": 2.77}

        try:
            # Run iPerf3 test
            cmd = "iperf3 -c 10.0.0.1 -t 10 -f M -J"
            result = subprocess.run(
                cmd, shell=True, capture_output=True, text=True, timeout=30
            )

            if result.returncode == 0:
                data = json.loads(result.stdout)
                actual_mbps = data["end"]["sum_received"]["bits_per_second"] / 1_000_000
                target = target_bandwidth[scenario]
                tolerance = target * 0.1  # 10% tolerance

                return {
                    "scenario": scenario,
                    "actual_mbps": actual_mbps,
                    "target_mbps": target,
                    "tolerance": tolerance,
                    "passed": abs(actual_mbps - target) <= tolerance,
                }

        except Exception as e:
            self.logger.warning(f"Throughput test failed for {scenario}: {e}")

        return {"scenario": scenario, "passed": False, "error": "Test execution failed"}

    def _validate_latency(self, scenario: str) -> Dict:
        """Validate latency for scenario"""
        target_latency = {"embb": 16.1, "urllc": 6.3, "miot": 15.7}

        try:
            # Run ping test
            cmd = "ping -c 100 -i 0.01 10.0.0.1"
            result = subprocess.run(
                cmd, shell=True, capture_output=True, text=True, timeout=30
            )

            if result.returncode == 0:
                # Parse ping output for average latency
                lines = result.stdout.split("\n")
                for line in lines:
                    if "avg" in line and "ms" in line:
                        parts = line.split("/")
                        if len(parts) >= 5:
                            avg_latency = float(parts[4])
                            target = target_latency[scenario]
                            tolerance = 2.0  # 2ms tolerance

                            return {
                                "scenario": scenario,
                                "actual_ms": avg_latency,
                                "target_ms": target,
                                "tolerance": tolerance,
                                "passed": abs(avg_latency - target) <= tolerance,
                            }

        except Exception as e:
            self.logger.warning(f"Latency test failed for {scenario}: {e}")

        return {"scenario": scenario, "passed": False, "error": "Test execution failed"}

    def _validate_resource_usage(self, results: Dict) -> Dict:
        """Validate resource usage against thresholds"""
        validation = {"passed": True, "details": {}}

        # Extract resource usage from results
        for scenario, data in results["scenarios"].items():
            metrics = data.get("metrics", {})
            resource_usage = metrics.get("resource_usage", {})

            # Validate SMO CPU usage
            smo_cpu = resource_usage.get("smo_cpu_peak", 0)
            cpu_limit = (
                self.thresholds.get("resource_limits", {})
                .get("smo", {})
                .get("cpu_max_cores", 2.0)
            )

            # Validate SMO memory usage
            smo_memory = resource_usage.get("smo_memory_peak", 0)
            memory_limit = (
                self.thresholds.get("resource_limits", {})
                .get("smo", {})
                .get("memory_max_mb", 4096)
            )

            validation["details"][scenario] = {
                "cpu": {
                    "actual": smo_cpu,
                    "limit": cpu_limit,
                    "passed": smo_cpu <= cpu_limit,
                },
                "memory": {
                    "actual": smo_memory,
                    "limit": memory_limit,
                    "passed": smo_memory <= memory_limit,
                },
            }

            # Check if any validation failed
            if not all(v["passed"] for v in validation["details"][scenario].values()):
                validation["passed"] = False

        return validation

    def generate_test_report(self, results: Dict, performance: Dict) -> Dict:
        """Generate comprehensive test report"""
        report = {
            "test_summary": {
                "series": results["series"],
                "total_duration": results["total_duration"],
                "scenarios_tested": len(results["scenarios"]),
                "deployment_passed": results["validation"]["passed"],
                "performance_passed": performance["passed"],
                "overall_passed": results["validation"]["passed"]
                and performance["passed"],
            },
            "deployment_results": results,
            "performance_validation": performance,
            "timestamp": time.time(),
            "test_harness_version": "1.0.0",
        }

        return report

    def save_report(self, report: Dict, filename: Optional[str] = None) -> str:
        """Save test report to file"""
        if not filename:
            timestamp = int(time.time())
            filename = (
                f"test_report_{report['test_summary']['series']}_{timestamp}.json"
            )

        self.results_dir.mkdir(exist_ok=True)
        report_path = self.results_dir / filename

        with open(report_path, "w") as f:
            json.dump(report, f, indent=2)

        self.logger.info(f"Test report saved to: {report_path}")
        return str(report_path)

    def _run_command(self, cmd: str, description: str, fail_ok: bool = False) -> bool:
        """Run a shell command and return success status"""
        self.logger.debug(f"Running: {cmd}")

        try:
            result = subprocess.run(
                cmd, shell=True, capture_output=True, text=True, timeout=300
            )

            if result.returncode == 0:
                self.logger.debug(f"✓ {description} - OK")
                return True
            else:
                if not fail_ok:
                    self.logger.error(f"✗ {description} - FAILED")
                    self.logger.error(f"Error: {result.stderr}")
                return False

        except subprocess.TimeoutExpired:
            self.logger.error(f"✗ {description} - TIMEOUT")
            return False
        except Exception as e:
            self.logger.error(f"✗ {description} - ERROR: {e}")
            return False


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="E2E Test Harness for O-RAN MANO")
    parser.add_argument(
        "--series",
        choices=["fast", "slow"],
        default="fast",
        help="Deployment series to test",
    )
    parser.add_argument(
        "--config-dir", default="config", help="Configuration directory"
    )
    parser.add_argument(
        "--results-dir", default="results", help="Results output directory"
    )
    parser.add_argument(
        "--skip-prereqs", action="store_true", help="Skip prerequisite checks"
    )
    parser.add_argument(
        "--skip-performance", action="store_true", help="Skip performance validation"
    )
    parser.add_argument("--output", help="Output report filename")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose logging")

    args = parser.parse_args()

    # Setup logging level
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    # Initialize test harness
    harness = TestHarness(args.config_dir, args.results_dir)

    try:
        # Run prerequisite checks
        if not args.skip_prereqs:
            if not harness.run_prerequisite_checks():
                harness.logger.error("Prerequisite checks failed")
                return 1

        # Run deployment tests
        deployment_results = harness.run_deployment_test(args.series)

        # Run performance validation
        performance_results = {}
        if not args.skip_performance:
            performance_results = harness.run_performance_validation(deployment_results)
        else:
            performance_results = {"passed": True, "skipped": True}

        # Generate and save report
        report = harness.generate_test_report(deployment_results, performance_results)
        report_path = harness.save_report(report, args.output)

        # Print summary
        summary = report["test_summary"]
        print("\n=== Test Summary ===")
        print(f"Series: {summary['series']}")
        print(f"Total Duration: {summary['total_duration']:.1f}s")
        print(f"Scenarios Tested: {summary['scenarios_tested']}")
        print(f"Deployment Passed: {'✓' if summary['deployment_passed'] else '✗'}")
        print(f"Performance Passed: {'✓' if summary['performance_passed'] else '✗'}")
        print(
            f"Overall Result: {'✓ PASSED' if summary['overall_passed'] else '✗ FAILED'}"
        )
        print(f"Report: {report_path}")

        return 0 if summary["overall_passed"] else 1

    except KeyboardInterrupt:
        harness.logger.info("Test interrupted by user")
        return 130
    except Exception as e:
        harness.logger.error(f"Test harness failed: {e}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
