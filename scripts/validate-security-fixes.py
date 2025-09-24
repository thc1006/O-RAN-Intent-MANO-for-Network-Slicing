#!/usr/bin/env python3
"""
Kubernetes Security Fixes Validation Script

This script validates that the security fixes applied to orchestrator.yaml and vnf-operator.yaml
address the Checkov compliance issues mentioned in the requirements.
"""

import sys
from pathlib import Path

import yaml


def load_yaml_docs(file_path):
    """Load all YAML documents from a file."""
    with open(file_path, "r") as f:
        return list(yaml.safe_load_all(f))


def check_service_account_token(docs, component_name):
    """Check if automountServiceAccountToken is properly configured."""
    deployment = next((doc for doc in docs if doc.get("kind") == "Deployment"), None)
    if not deployment:
        return False, f"No Deployment found for {component_name}"

    spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
    token_mount = spec.get("automountServiceAccountToken")

    if token_mount is True:
        return (
            True,
            f"PASS {component_name}: automountServiceAccountToken correctly set to true",
        )
    else:
        return (
            False,
            f"FAIL {component_name}: automountServiceAccountToken should be true, got {token_mount}",
        )


def check_image_reference(docs, component_name):
    """Check if image references use proper versioned tags."""
    deployment = next((doc for doc in docs if doc.get("kind") == "Deployment"), None)
    if not deployment:
        return False, f"No Deployment found for {component_name}"

    containers = (
        deployment.get("spec", {})
        .get("template", {})
        .get("spec", {})
        .get("containers", [])
    )
    if not containers:
        return False, f"No containers found for {component_name}"

    image = containers[0].get("image", "")

    # Check for proper versioned tag (without placeholder digest)
    if ":v1.0.1" in image and "@sha256:" not in image:
        return (
            True,
            f"PASS {component_name}: Using proper versioned tag without placeholder digest",
        )
    elif "@sha256:" in image and len(image.split("@sha256:")[1]) == 64:
        return True, f"PASS {component_name}: Using proper SHA256 digest"
    else:
        return (
            False,
            f"FAIL {component_name}: Image reference needs proper versioning: {image}",
        )


def check_image_pull_policy(docs, component_name):
    """Check if imagePullPolicy is set to Always."""
    deployment = next((doc for doc in docs if doc.get("kind") == "Deployment"), None)
    if not deployment:
        return False, f"No Deployment found for {component_name}"

    containers = (
        deployment.get("spec", {})
        .get("template", {})
        .get("spec", {})
        .get("containers", [])
    )
    if not containers:
        return False, f"No containers found for {component_name}"

    pull_policy = containers[0].get("imagePullPolicy")

    if pull_policy == "Always":
        return True, f"PASS {component_name}: imagePullPolicy correctly set to Always"
    else:
        return (
            False,
            f"FAIL {component_name}: imagePullPolicy should be Always, got {pull_policy}",
        )


def check_seccomp_profile(docs, component_name):
    """Check if seccomp profiles are properly configured."""
    deployment = next((doc for doc in docs if doc.get("kind") == "Deployment"), None)
    if not deployment:
        return False, f"No Deployment found for {component_name}"

    # Check pod-level seccomp
    pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
    pod_seccomp = (
        pod_spec.get("securityContext", {}).get("seccompProfile", {}).get("type")
    )

    # Check container-level seccomp
    containers = pod_spec.get("containers", [])
    if not containers:
        return False, f"No containers found for {component_name}"

    container_seccomp = (
        containers[0].get("securityContext", {}).get("seccompProfile", {}).get("type")
    )

    if pod_seccomp == "RuntimeDefault" and container_seccomp == "RuntimeDefault":
        return (
            True,
            f"PASS {component_name}: Seccomp profiles correctly set to RuntimeDefault",
        )
    else:
        return (
            False,
            f"FAIL {component_name}: Seccomp profiles should be RuntimeDefault, got pod:{pod_seccomp}, container:{container_seccomp}",
        )


def check_security_annotations(docs, component_name):
    """Check if security annotations are present."""
    deployment = next((doc for doc in docs if doc.get("kind") == "Deployment"), None)
    if not deployment:
        return False, f"No Deployment found for {component_name}"

    # Check for security justification annotations
    annotations = deployment.get("metadata", {}).get("annotations", {})
    pod_annotations = (
        deployment.get("spec", {})
        .get("template", {})
        .get("metadata", {})
        .get("annotations", {})
    )

    has_security_annotations = any(
        [
            "seccomp.security.alpha.kubernetes.io/pod" in annotations,
            "seccomp.security.alpha.kubernetes.io/pod" in pod_annotations,
            "security.policy" in str(annotations),
            "security.compliance" in str(pod_annotations),
        ]
    )

    if has_security_annotations:
        return True, f"PASS {component_name}: Security annotations present"
    else:
        return False, f"FAIL {component_name}: Missing security annotations"


def check_rbac_configuration(rbac_docs):
    """Check RBAC service account configurations."""
    service_accounts = [doc for doc in rbac_docs if doc.get("kind") == "ServiceAccount"]

    results = []
    for sa in service_accounts:
        name = sa.get("metadata", {}).get("name", "unknown")
        if name in ["oran-orchestrator", "oran-vnf-operator"]:
            token_mount = sa.get("automountServiceAccountToken")
            annotations = sa.get("metadata", {}).get("annotations", {})

            if token_mount is True:
                results.append(
                    (
                        True,
                        f"PASS ServiceAccount {name}: automountServiceAccountToken correctly set to true",
                    )
                )
            else:
                results.append(
                    (
                        False,
                        f"FAIL ServiceAccount {name}: automountServiceAccountToken should be true",
                    )
                )

            if "security.policy/api-access-reason" in annotations:
                results.append(
                    (
                        True,
                        f"PASS ServiceAccount {name}: Security justification annotation present",
                    )
                )
            else:
                results.append(
                    (
                        False,
                        f"FAIL ServiceAccount {name}: Missing security justification annotation",
                    )
                )

    return results


def main():
    """Main validation function."""
    script_dir = Path(__file__).parent
    base_dir = script_dir.parent / "deploy" / "k8s" / "base"

    orchestrator_file = base_dir / "orchestrator.yaml"
    vnf_operator_file = base_dir / "vnf-operator.yaml"
    rbac_file = base_dir / "rbac.yaml"

    if not all(
        [orchestrator_file.exists(), vnf_operator_file.exists(), rbac_file.exists()]
    ):
        print("FAIL Required YAML files not found")
        return 1

    print("Kubernetes Security Fixes Validation")
    print("=" * 50)

    all_passed = True

    # Check orchestrator
    print("\nCHECKING Orchestrator Component:")
    orchestrator_docs = load_yaml_docs(orchestrator_file)

    checks = [
        check_service_account_token(orchestrator_docs, "orchestrator"),
        check_image_reference(orchestrator_docs, "orchestrator"),
        check_image_pull_policy(orchestrator_docs, "orchestrator"),
        check_seccomp_profile(orchestrator_docs, "orchestrator"),
        check_security_annotations(orchestrator_docs, "orchestrator"),
    ]

    for passed, message in checks:
        print(f"  {message}")
        if not passed:
            all_passed = False

    # Check VNF operator
    print("\nCHECKING VNF Operator Component:")
    vnf_operator_docs = load_yaml_docs(vnf_operator_file)

    checks = [
        check_service_account_token(vnf_operator_docs, "vnf-operator"),
        check_image_reference(vnf_operator_docs, "vnf-operator"),
        check_image_pull_policy(vnf_operator_docs, "vnf-operator"),
        check_seccomp_profile(vnf_operator_docs, "vnf-operator"),
        check_security_annotations(vnf_operator_docs, "vnf-operator"),
    ]

    for passed, message in checks:
        print(f"  {message}")
        if not passed:
            all_passed = False

    # Check RBAC
    print("\nCHECKING RBAC Configuration:")
    rbac_docs = load_yaml_docs(rbac_file)
    rbac_results = check_rbac_configuration(rbac_docs)

    for passed, message in rbac_results:
        print(f"  {message}")
        if not passed:
            all_passed = False

    print("\n" + "=" * 50)
    if all_passed:
        print("SUCCESS All security checks passed!")
        return 0
    else:
        print("WARNING  Some security checks failed. Please review the issues above.")
        return 1


if __name__ == "__main__":
    sys.exit(main())
