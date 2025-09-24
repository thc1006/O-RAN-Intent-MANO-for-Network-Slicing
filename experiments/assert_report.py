#!/usr/bin/env python3
"""
O-RAN Intent MANO - Report Assertion Module

Validates metrics report against thesis performance thresholds:
- Deploy time < 10 minutes
- DL throughput within ±10% of targets: {4.57, 2.77, 0.93} Mbps
- Ping RTT within ±10% of targets: {16.1, 15.7, 6.3} ms

Exit codes:
- 0: All assertions passed
- 1: One or more assertions failed
- 2: Invalid input or system error
"""

import argparse
import json
import logging
import sys
from pathlib import Path
from typing import Any, Dict


def setup_logging(verbose: bool = False) -> None:
    """Configure logging."""
    level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(
        level=level,
        format="%(asctime)s - %(levelname)s - %(message)s",
        datefmt="%H:%M:%S",
    )


def load_report(report_file: Path) -> Dict[str, Any]:
    """Load JSON report from file."""
    if not report_file.exists():
        raise FileNotFoundError(f"Report file not found: {report_file}")

    try:
        with open(report_file, "r") as f:
            return json.load(f)
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON in report file: {e}")


def assert_overall_status(report: Dict[str, Any]) -> bool:
    """Assert that overall test status is PASS."""
    overall_status = report.get("overall_status")

    if overall_status == "PASS":
        logging.info("✅ Overall status: PASS")
        return True
    else:
        logging.error(f"❌ Overall status: {overall_status}")
        return False


def assert_deployment_time(report: Dict[str, Any], max_time_minutes: int = 10) -> bool:
    """Assert that all deployment times are under the specified limit."""
    max_time_seconds = max_time_minutes * 60
    all_passed = True

    profiles = report.get("profiles", {})

    for profile_name, profile_data in profiles.items():
        metrics = profile_data.get("metrics", {})
        deploy_time = metrics.get("deployment_time")

        if not deploy_time:
            logging.error(f"❌ {profile_name}: No deployment time data")
            all_passed = False
            continue

        actual_time = deploy_time.get("actual_s", 0)
        within_limit = deploy_time.get("within_limit", False)

        if within_limit and actual_time <= max_time_seconds:
            logging.info(
                f"✅ {profile_name}: Deployment time {actual_time}s < {max_time_seconds}s"
            )
        else:
            logging.error(
                f"❌ {profile_name}: Deployment time {actual_time}s >= {max_time_seconds}s"
            )
            all_passed = False

    return all_passed


def assert_bandwidth_targets(
    report: Dict[str, Any], tolerance_percent: float = 10.0
) -> bool:
    """Assert that all bandwidth measurements are within tolerance of targets."""
    all_passed = True

    profiles = report.get("profiles", {})

    for profile_name, profile_data in profiles.items():
        metrics = profile_data.get("metrics", {})
        bandwidth = metrics.get("bandwidth")

        if not bandwidth:
            logging.error(f"❌ {profile_name}: No bandwidth data")
            all_passed = False
            continue

        within_tolerance = bandwidth.get("within_tolerance", False)
        actual_mbps = bandwidth.get("actual_mbps", 0)
        expected_mbps = bandwidth.get("expected_mbps", 0)
        deviation_percent = bandwidth.get("deviation_percent", 0)

        if within_tolerance:
            logging.info(
                f"✅ {profile_name}: Bandwidth {actual_mbps:.2f}Mbps "
                f"(expected: {expected_mbps:.2f}Mbps, deviation: {deviation_percent:.1f}%)"
            )
        else:
            logging.error(
                f"❌ {profile_name}: Bandwidth {actual_mbps:.2f}Mbps "
                f"(expected: {expected_mbps:.2f}Mbps, deviation: {deviation_percent:.1f}%)"
            )
            all_passed = False

    return all_passed


def assert_latency_targets(
    report: Dict[str, Any], tolerance_percent: float = 10.0
) -> bool:
    """Assert that all latency measurements are within tolerance of targets."""
    all_passed = True

    profiles = report.get("profiles", {})

    for profile_name, profile_data in profiles.items():
        metrics = profile_data.get("metrics", {})
        latency = metrics.get("latency")

        if not latency:
            logging.error(f"❌ {profile_name}: No latency data")
            all_passed = False
            continue

        within_tolerance = latency.get("within_tolerance", False)
        actual_ms = latency.get("actual_ms", 0)
        expected_ms = latency.get("expected_ms", 0)
        deviation_percent = latency.get("deviation_percent", 0)

        if within_tolerance:
            logging.info(
                f"✅ {profile_name}: Latency {actual_ms:.1f}ms "
                f"(expected: {expected_ms:.1f}ms, deviation: {deviation_percent:.1f}%)"
            )
        else:
            logging.error(
                f"❌ {profile_name}: Latency {actual_ms:.1f}ms "
                f"(expected: {expected_ms:.1f}ms, deviation: {deviation_percent:.1f}%)"
            )
            all_passed = False

    return all_passed


def assert_success_rate(
    report: Dict[str, Any], min_success_rate: float = 100.0
) -> bool:
    """Assert that success rate meets minimum threshold."""
    summary = report.get("summary", {})
    success_rate = summary.get("success_rate_percent", 0.0)

    if success_rate >= min_success_rate:
        logging.info(f"✅ Success rate: {success_rate}% >= {min_success_rate}%")
        return True
    else:
        logging.error(f"❌ Success rate: {success_rate}% < {min_success_rate}%")
        return False


def assert_no_critical_issues(report: Dict[str, Any]) -> bool:
    """Assert that there are no critical issues."""
    summary = report.get("summary", {})
    critical_issues = summary.get("critical_issues", [])

    if not critical_issues:
        logging.info("✅ No critical issues found")
        return True
    else:
        logging.error(f"❌ Found {len(critical_issues)} critical issues:")
        for issue in critical_issues:
            logging.error(f"   - {issue}")
        return False


def run_all_assertions(report: Dict[str, Any], args: argparse.Namespace) -> bool:
    """Run all assertions and return overall result."""
    logging.info("Running performance threshold assertions...")

    assertions = [
        ("Overall Status", lambda: assert_overall_status(report)),
        (
            "Deployment Time",
            lambda: assert_deployment_time(report, args.max_deploy_minutes),
        ),
        ("Bandwidth Targets", lambda: assert_bandwidth_targets(report, args.tolerance)),
        ("Latency Targets", lambda: assert_latency_targets(report, args.tolerance)),
        ("Success Rate", lambda: assert_success_rate(report, args.min_success_rate)),
        ("Critical Issues", lambda: assert_no_critical_issues(report)),
    ]

    results = []

    for assertion_name, assertion_func in assertions:
        logging.info(f"\n--- Asserting: {assertion_name} ---")
        try:
            result = assertion_func()
            results.append(result)

            if result:
                logging.info(f"✅ {assertion_name}: PASSED")
            else:
                logging.error(f"❌ {assertion_name}: FAILED")

        except Exception as e:
            logging.error(f"❌ {assertion_name}: ERROR - {e}")
            results.append(False)

    all_passed = all(results)

    # Summary
    passed_count = sum(results)
    total_count = len(results)

    logging.info(f"\n{'='*50}")
    logging.info(f"ASSERTION SUMMARY: {passed_count}/{total_count} passed")

    if all_passed:
        logging.info("🎉 All assertions PASSED! System meets performance targets.")
    else:
        logging.error("💥 One or more assertions FAILED! System does not meet targets.")

    logging.info(f"{'='*50}")

    return all_passed


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Assert O-RAN Intent MANO performance thresholds"
    )
    parser.add_argument(
        "report_file", type=Path, help="Path to JSON metrics report file"
    )
    parser.add_argument(
        "--tolerance",
        type=float,
        default=10.0,
        help="Tolerance percentage for bandwidth/latency (default: 10.0)",
    )
    parser.add_argument(
        "--max-deploy-minutes",
        type=int,
        default=10,
        help="Maximum deployment time in minutes (default: 10)",
    )
    parser.add_argument(
        "--min-success-rate",
        type=float,
        default=100.0,
        help="Minimum success rate percentage (default: 100.0)",
    )
    parser.add_argument(
        "--verbose", "-v", action="store_true", help="Enable verbose logging"
    )

    args = parser.parse_args()

    setup_logging(args.verbose)

    try:
        # Load report
        logging.info(f"Loading report from: {args.report_file}")
        report = load_report(args.report_file)

        # Run assertions
        all_passed = run_all_assertions(report, args)

        # Exit with appropriate code
        if all_passed:
            sys.exit(0)  # Success
        else:
            sys.exit(1)  # Assertion failure

    except FileNotFoundError as e:
        logging.error(f"File not found: {e}")
        sys.exit(2)  # System error
    except ValueError as e:
        logging.error(f"Invalid data: {e}")
        sys.exit(2)  # System error
    except Exception as e:
        logging.error(f"Unexpected error: {e}")
        sys.exit(2)  # System error


if __name__ == "__main__":
    main()
