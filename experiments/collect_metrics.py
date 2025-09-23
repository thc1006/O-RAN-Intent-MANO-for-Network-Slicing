#!/usr/bin/env python3
"""
E2E Deployment Metrics Collection and Analysis
Collects timing, resource usage, and bottleneck metrics
"""

import json
import time
import subprocess
import argparse
import sys
import threading
import statistics
from datetime import datetime
from collections import defaultdict
from typing import Dict, List, Tuple, Any

class MetricsCollector:
    """Collects and analyzes deployment metrics"""

    def __init__(self):
        self.metrics = {
            "timestamp": datetime.now().isoformat(),
            "scenarios": {},
            "resources": {
                "smo": {"cpu": [], "memory": []},
                "ocloud": {"memory_per_node": {}},
                "pods": {"count": [], "states": {}}
            },
            "bottlenecks": {
                "smf_timeline": [],
                "smf_init_delay": 0,
                "porch_sync_times": [],
                "configsync_latencies": []
            },
            "timing": {},
            "validation": {
                "target_series": "unknown",
                "deviations": {},
                "pass_fail": {}
            }
        }
        self.collection_threads = []
        self.stop_collection = False

    def run_kubectl(self, cmd: str) -> str:
        """Execute kubectl command and return output"""
        try:
            result = subprocess.run(
                f"kubectl {cmd}",
                shell=True,
                capture_output=True,
                text=True,
                timeout=10
            )
            return result.stdout.strip()
        except subprocess.TimeoutExpired:
            return ""
        except Exception as e:
            print(f"Error running kubectl: {e}", file=sys.stderr)
            return ""

    def collect_smo_metrics(self) -> Dict[str, float]:
        """Collect SMO (Service Management and Orchestration) metrics"""
        metrics = {"cpu": 0.0, "memory": 0.0}

        # Get SMO pods (assuming they're in oran-system namespace)
        pods_json = self.run_kubectl(
            "get pods -n oran-system -o json"
        )

        if not pods_json:
            return metrics

        try:
            pods = json.loads(pods_json)
            total_cpu = 0.0
            total_memory = 0.0

            for pod in pods.get("items", []):
                pod_name = pod["metadata"]["name"]

                # Get pod metrics
                metrics_output = self.run_kubectl(
                    f"top pod {pod_name} -n oran-system --no-headers"
                )

                if metrics_output:
                    parts = metrics_output.split()
                    if len(parts) >= 3:
                        # Parse CPU (remove 'm' suffix and convert to cores)
                        cpu_str = parts[1].replace('m', '')
                        total_cpu += float(cpu_str) / 1000.0

                        # Parse memory (convert to MB)
                        mem_str = parts[2].replace('Mi', '').replace('Gi', '000')
                        total_memory += float(mem_str)

            metrics["cpu"] = round(total_cpu, 3)
            metrics["memory"] = round(total_memory, 2)

        except (json.JSONDecodeError, KeyError, ValueError) as e:
            print(f"Error parsing metrics: {e}", file=sys.stderr)

        return metrics

    def collect_ocloud_memory(self, nodes: List[str]) -> Dict[str, float]:
        """Collect O-Cloud memory usage per node"""
        memory_usage = {}

        for node in nodes:
            # Get node metrics
            metrics_output = self.run_kubectl(
                f"top node {node} --no-headers"
            )

            if metrics_output:
                parts = metrics_output.split()
                if len(parts) >= 5:
                    # Parse memory percentage
                    mem_percent = parts[4].replace('%', '')

                    # Get node capacity
                    node_json = self.run_kubectl(
                        f"get node {node} -o json"
                    )

                    if node_json:
                        try:
                            node_data = json.loads(node_json)
                            mem_capacity = node_data["status"]["capacity"]["memory"]
                            # Convert Ki to MB
                            mem_capacity_mb = int(mem_capacity.replace('Ki', '')) / 1024
                            mem_used_mb = mem_capacity_mb * float(mem_percent) / 100
                            memory_usage[node] = round(mem_used_mb, 2)
                        except (json.JSONDecodeError, KeyError) as e:
                            print(f"Error getting node capacity: {e}", file=sys.stderr)

        return memory_usage

    def monitor_smf_bottleneck(self, scenario: str) -> None:
        """Monitor SMF pod for initialization bottleneck"""
        smf_pod = f"cn-{scenario}"
        namespace = "oran-system"
        timeline = []
        start_time = time.time()

        # Initial state
        timeline.append({
            "t": 0,
            "event": "Monitoring started",
            "cpu": 0,
            "state": "Pending"
        })

        # Monitor for 3 minutes
        while time.time() - start_time < 180:
            if self.stop_collection:
                break

            # Get pod status
            pod_json = self.run_kubectl(
                f"get pod -n {namespace} -l scenario={scenario},app=smf -o json"
            )

            if pod_json:
                try:
                    pods = json.loads(pod_json)
                    if pods["items"]:
                        pod = pods["items"][0]
                        phase = pod["status"]["phase"]

                        # Get CPU usage
                        metrics = self.run_kubectl(
                            f"top pod {pod['metadata']['name']} -n {namespace} --no-headers"
                        )

                        cpu_usage = 0
                        if metrics:
                            parts = metrics.split()
                            if len(parts) >= 2:
                                cpu_usage = float(parts[1].replace('m', '')) / 1000.0

                        # Check for bottleneck pattern (high CPU during init)
                        elapsed = time.time() - start_time

                        # Record significant events
                        if phase == "Running" and cpu_usage > 0.8:  # 80% CPU
                            timeline.append({
                                "t": elapsed,
                                "event": "High CPU detected - SMF initialization",
                                "cpu": cpu_usage,
                                "state": phase
                            })

                            # This is the bottleneck
                            if self.metrics["bottlenecks"]["smf_init_delay"] == 0:
                                self.metrics["bottlenecks"]["smf_init_delay"] = elapsed

                        # Record ready state
                        if phase == "Running":
                            conditions = pod["status"].get("conditions", [])
                            for cond in conditions:
                                if cond["type"] == "Ready" and cond["status"] == "True":
                                    timeline.append({
                                        "t": elapsed,
                                        "event": "SMF Ready",
                                        "cpu": cpu_usage,
                                        "state": "Ready"
                                    })
                                    self.metrics["bottlenecks"]["smf_timeline"] = timeline
                                    return

                except (json.JSONDecodeError, KeyError) as e:
                    print(f"Error monitoring SMF: {e}", file=sys.stderr)

            time.sleep(2)

        self.metrics["bottlenecks"]["smf_timeline"] = timeline

    def collect_continuous_metrics(self, interval: int = 5) -> None:
        """Continuously collect metrics at specified interval"""
        while not self.stop_collection:
            # Collect SMO metrics
            smo_metrics = self.collect_smo_metrics()
            self.metrics["resources"]["smo"]["cpu"].append(smo_metrics["cpu"])
            self.metrics["resources"]["smo"]["memory"].append(smo_metrics["memory"])

            # Collect pod count
            pods_output = self.run_kubectl(
                "get pods --all-namespaces -l experiment=e2e-test -o json"
            )

            if pods_output:
                try:
                    pods = json.loads(pods_output)
                    self.metrics["resources"]["pods"]["count"].append(len(pods["items"]))
                except json.JSONDecodeError:
                    pass

            time.sleep(interval)

    def process_intent(self, scenario: str, input_file: str, output_file: str) -> None:
        """Simulate intent processing and generate configuration"""
        print(f"Processing intent for scenario: {scenario}")

        # Simulate NLP processing delay
        time.sleep(2)

        # Generate configuration based on scenario
        config = {
            "scenario": scenario,
            "timestamp": datetime.now().isoformat(),
            "qos": {},
            "placement": {}
        }

        if scenario == "embb":
            config["qos"] = {
                "bandwidth": 4.57,
                "latency": 16.1,
                "jitter": 2.0,
                "reliability": 99.9
            }
            config["placement"] = {"type": "edge", "zones": ["edge-1", "edge-2"]}
        elif scenario == "urllc":
            config["qos"] = {
                "bandwidth": 0.93,
                "latency": 6.3,
                "jitter": 0.5,
                "reliability": 99.999
            }
            config["placement"] = {"type": "edge", "zones": ["edge-1"]}
        elif scenario == "miot":
            config["qos"] = {
                "bandwidth": 2.77,
                "latency": 15.7,
                "jitter": 2.5,
                "reliability": 99.5
            }
            config["placement"] = {"type": "regional", "zones": ["regional-1"]}

        # Save configuration
        with open(output_file, 'w') as f:
            json.dump(config, f, indent=2)

        print(f"Configuration generated: {output_file}")

    def start_domain_collection(self, domain: str, scenario: str) -> None:
        """Start metrics collection for a specific domain"""
        print(f"Starting metrics collection for {domain} - {scenario}")

        # Initialize scenario metrics if not exists
        if scenario not in self.metrics["scenarios"]:
            self.metrics["scenarios"][scenario] = {
                "domains": {},
                "total_time": 0
            }

        # Record domain start time
        self.metrics["scenarios"][scenario]["domains"][domain] = {
            "start_time": time.time(),
            "end_time": 0,
            "duration": 0,
            "pod_count": 0
        }

    def validate_results(self, target_series: Dict[str, Dict]) -> Dict[str, Any]:
        """Validate results against target thresholds"""
        validation = {
            "series_type": "fast" if target_series.get("embb", {}).get("target", 0) < 500 else "slow",
            "passed": True,
            "failures": [],
            "details": {}
        }

        for scenario, targets in target_series.items():
            if scenario in self.metrics["scenarios"]:
                actual_time = self.metrics["scenarios"][scenario].get("total_time", 0)
                target_time = targets["target"]
                tolerance = targets["tolerance"]

                deviation = abs(actual_time - target_time)
                passed = deviation <= tolerance

                validation["details"][scenario] = {
                    "actual": actual_time,
                    "target": target_time,
                    "deviation": deviation,
                    "tolerance": tolerance,
                    "passed": passed
                }

                if not passed:
                    validation["passed"] = False
                    validation["failures"].append(
                        f"{scenario}: {actual_time}s (target: {target_time}±{tolerance}s)"
                    )

        # Check SMF bottleneck
        smf_delay = self.metrics["bottlenecks"]["smf_init_delay"]
        if smf_delay > 60:
            validation["smf_bottleneck_detected"] = True
            validation["smf_delay"] = smf_delay

        # Check resource usage
        if self.metrics["resources"]["smo"]["cpu"]:
            peak_cpu = max(self.metrics["resources"]["smo"]["cpu"])
            validation["smo_cpu_peak"] = peak_cpu
            if peak_cpu > 2.0:  # 2 cores threshold
                validation["passed"] = False
                validation["failures"].append(f"SMO CPU exceeded: {peak_cpu} cores")

        return validation

    def generate_report(self, output_file: str, html_file: str = None) -> None:
        """Generate final JSON report and optional HTML visualization"""

        # Calculate statistics
        if self.metrics["resources"]["smo"]["cpu"]:
            self.metrics["resources"]["smo"]["cpu_peak"] = max(self.metrics["resources"]["smo"]["cpu"])
            self.metrics["resources"]["smo"]["cpu_avg"] = statistics.mean(self.metrics["resources"]["smo"]["cpu"])

        if self.metrics["resources"]["smo"]["memory"]:
            self.metrics["resources"]["smo"]["memory_peak"] = max(self.metrics["resources"]["smo"]["memory"])
            self.metrics["resources"]["smo"]["memory_avg"] = statistics.mean(self.metrics["resources"]["smo"]["memory"])

        # Target series for validation
        target_series = {
            "embb": {"target": 407, "tolerance": 20},
            "urllc": {"target": 353, "tolerance": 20},
            "miot": {"target": 257, "tolerance": 20}
        }

        # Validate results
        self.metrics["validation"] = self.validate_results(target_series)

        # Save JSON report
        with open(output_file, 'w') as f:
            json.dump(self.metrics, f, indent=2)

        print(f"JSON report saved: {output_file}")

        # Generate HTML report if requested
        if html_file:
            self.generate_html_report(html_file)

    def generate_html_report(self, html_file: str) -> None:
        """Generate HTML visualization of metrics"""
        html_content = """
<!DOCTYPE html>
<html>
<head>
    <title>E2E Deployment Metrics Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .metric-card {
            border: 1px solid #ddd;
            padding: 15px;
            margin: 10px 0;
            border-radius: 5px;
        }
        .pass { background-color: #d4edda; }
        .fail { background-color: #f8d7da; }
        .timeline {
            border-left: 3px solid #007bff;
            padding-left: 20px;
            margin: 20px 0;
        }
        .timeline-event {
            margin: 10px 0;
            padding: 5px;
            background-color: #f0f0f0;
        }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>E2E Deployment Metrics Report</h1>
    <p>Generated: {timestamp}</p>

    <div class="metric-card {validation_class}">
        <h2>Validation Result: {validation_status}</h2>
        <p>Target Series: {series_type}</p>
        {validation_details}
    </div>

    <div class="metric-card">
        <h2>Deployment Times</h2>
        <table>
            <tr>
                <th>Scenario</th>
                <th>Actual (s)</th>
                <th>Target (s)</th>
                <th>Deviation (%)</th>
                <th>Status</th>
            </tr>
            {timing_rows}
        </table>
    </div>

    <div class="metric-card">
        <h2>SMF Bottleneck Analysis</h2>
        <p>Initialization Delay: {smf_delay}s</p>
        <div class="timeline">
            {smf_timeline}
        </div>
    </div>

    <div class="metric-card">
        <h2>Resource Usage</h2>
        <h3>SMO Metrics</h3>
        <ul>
            <li>CPU Peak: {smo_cpu_peak} cores</li>
            <li>CPU Average: {smo_cpu_avg} cores</li>
            <li>Memory Peak: {smo_mem_peak} MB</li>
            <li>Memory Average: {smo_mem_avg} MB</li>
        </ul>

        <h3>O-Cloud Memory</h3>
        <ul>
            {ocloud_memory}
        </ul>
    </div>
</body>
</html>
        """

        # Fill in template
        validation = self.metrics.get("validation", {})

        # Build timing rows
        timing_rows = ""
        for scenario in ["embb", "urllc", "miot"]:
            if scenario in validation.get("details", {}):
                detail = validation["details"][scenario]
                status = "✓" if detail["passed"] else "✗"
                deviation_pct = (detail["deviation"] / detail["target"]) * 100
                timing_rows += f"""
                <tr>
                    <td>{scenario.upper()}</td>
                    <td>{detail['actual']:.1f}</td>
                    <td>{detail['target']}</td>
                    <td>{deviation_pct:.1f}%</td>
                    <td>{status}</td>
                </tr>
                """

        # Build SMF timeline
        smf_timeline = ""
        for event in self.metrics["bottlenecks"].get("smf_timeline", []):
            smf_timeline += f"""
            <div class="timeline-event">
                <strong>T+{event['t']:.1f}s:</strong> {event['event']}
                (CPU: {event.get('cpu', 0):.2f} cores)
            </div>
            """

        # Build O-Cloud memory list
        ocloud_memory = ""
        for node, mem in self.metrics["resources"]["ocloud"].get("memory_per_node", {}).items():
            ocloud_memory += f"<li>{node}: {mem} MB</li>"

        # Fill template
        html_filled = html_content.format(
            timestamp=self.metrics["timestamp"],
            validation_class="pass" if validation.get("passed") else "fail",
            validation_status="PASSED" if validation.get("passed") else "FAILED",
            series_type=validation.get("series_type", "unknown"),
            validation_details=", ".join(validation.get("failures", [])) or "All checks passed",
            timing_rows=timing_rows,
            smf_delay=self.metrics["bottlenecks"].get("smf_init_delay", 0),
            smf_timeline=smf_timeline or "<p>No bottleneck detected</p>",
            smo_cpu_peak=self.metrics["resources"]["smo"].get("cpu_peak", 0),
            smo_cpu_avg=self.metrics["resources"]["smo"].get("cpu_avg", 0),
            smo_mem_peak=self.metrics["resources"]["smo"].get("memory_peak", 0),
            smo_mem_avg=self.metrics["resources"]["smo"].get("memory_avg", 0),
            ocloud_memory=ocloud_memory or "<li>No data</li>"
        )

        with open(html_file, 'w') as f:
            f.write(html_filled)

        print(f"HTML report saved: {html_file}")

def main():
    parser = argparse.ArgumentParser(description="E2E Deployment Metrics Collection")

    subparsers = parser.add_subparsers(dest="command", help="Command to run")

    # Process intent command
    intent_parser = subparsers.add_parser("process_intent", help="Process intent")
    intent_parser.add_argument("--scenario", required=True, help="Scenario name")
    intent_parser.add_argument("--input", required=True, help="Input file")
    intent_parser.add_argument("--output", required=True, help="Output file")

    # Start collection command
    collect_parser = subparsers.add_parser("start_collection", help="Start metrics collection")
    collect_parser.add_argument("--domain", required=True, help="Domain (ran/tn/cn)")
    collect_parser.add_argument("--scenario", required=True, help="Scenario name")

    # Monitor SMF command
    smf_parser = subparsers.add_parser("monitor_smf", help="Monitor SMF bottleneck")
    smf_parser.add_argument("--scenario", required=True, help="Scenario name")

    # Continuous collection command
    continuous_parser = subparsers.add_parser("continuous", help="Continuous metrics collection")
    continuous_parser.add_argument("--output", required=True, help="Output file")
    continuous_parser.add_argument("--interval", type=int, default=5, help="Collection interval")

    # System metrics command
    system_parser = subparsers.add_parser("collect_system", help="Collect system metrics")
    system_parser.add_argument("--output", required=True, help="Output file")
    system_parser.add_argument("--smo-namespace", default="oran-system", help="SMO namespace")
    system_parser.add_argument("--ocloud-nodes", required=True, help="Comma-separated node names")

    # Generate report command
    report_parser = subparsers.add_parser("generate_report", help="Generate final report")
    report_parser.add_argument("--metrics", required=True, help="Metrics file")
    report_parser.add_argument("--timers", help="Timers file")
    report_parser.add_argument("--output", required=True, help="Output JSON file")
    report_parser.add_argument("--html", help="Output HTML file")

    args = parser.parse_args()

    collector = MetricsCollector()

    if args.command == "process_intent":
        collector.process_intent(args.scenario, args.input, args.output)

    elif args.command == "start_collection":
        collector.start_domain_collection(args.domain, args.scenario)

    elif args.command == "monitor_smf":
        collector.monitor_smf_bottleneck(args.scenario)

    elif args.command == "continuous":
        # Run continuous collection in background
        thread = threading.Thread(
            target=collector.collect_continuous_metrics,
            args=(args.interval,)
        )
        thread.daemon = True
        thread.start()

        # Keep running until interrupted
        try:
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            collector.stop_collection = True

        # Save metrics
        with open(args.output, 'w') as f:
            json.dump(collector.metrics, f, indent=2)

    elif args.command == "collect_system":
        # Collect system-wide metrics once
        smo_metrics = collector.collect_smo_metrics()
        nodes = args.ocloud_nodes.split(',')
        ocloud_memory = collector.collect_ocloud_memory(nodes)

        collector.metrics["resources"]["smo"]["final"] = smo_metrics
        collector.metrics["resources"]["ocloud"]["memory_per_node"] = ocloud_memory

        with open(args.output, 'w') as f:
            json.dump(collector.metrics, f, indent=2)

    elif args.command == "generate_report":
        # Load metrics
        if args.metrics:
            with open(args.metrics, 'r') as f:
                collector.metrics = json.load(f)

        # Generate report
        collector.generate_report(args.output, args.html)

if __name__ == "__main__":
    main()