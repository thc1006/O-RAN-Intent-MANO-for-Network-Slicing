#!/usr/bin/env python3
import re
import os

def fix_dot_imports(file_path):
    """Fix dot-imports in integration test files by replacing them with explicit package references."""

    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Step 1: Replace dot-imports with explicit imports
    content = content.replace(
        '. "github.com/onsi/ginkgo/v2"',
        '"github.com/onsi/ginkgo/v2"'
    )
    content = content.replace(
        '. "github.com/onsi/gomega"',
        '"github.com/onsi/gomega"'
    )

    # Step 2: Replace function calls with package prefixes
    # Ginkgo function calls
    content = re.sub(r'\bDescribe\b', 'ginkgo.Describe', content)
    content = re.sub(r'\bContext\b', 'ginkgo.Context', content)
    content = re.sub(r'\bIt\b', 'ginkgo.It', content)
    content = re.sub(r'\bBy\b', 'ginkgo.By', content)
    content = re.sub(r'\bBeforeEach\b', 'ginkgo.BeforeEach', content)
    content = re.sub(r'\bAfterEach\b', 'ginkgo.AfterEach', content)
    content = re.sub(r'\bGinkgoRecover\b', 'ginkgo.GinkgoRecover', content)
    content = re.sub(r'\bRegisterFailHandler\b', 'ginkgo.RegisterFailHandler', content)
    content = re.sub(r'\bRunSpecs\b', 'ginkgo.RunSpecs', content)
    content = re.sub(r'\bFail\b(?!\w)', 'ginkgo.Fail', content)  # Match Fail but not Failure/Failed

    # Gomega function calls
    content = re.sub(r'\bExpect\b', 'gomega.Expect', content)
    content = re.sub(r'\.To\(BeTrue\b', '.To(gomega.BeTrue', content)
    content = re.sub(r'\.To\(BeFalse\b', '.To(gomega.BeFalse', content)
    content = re.sub(r'\.To\(BeNumerically\b', '.To(gomega.BeNumerically', content)
    content = re.sub(r'\.To\(Equal\b', '.To(gomega.Equal', content)
    content = re.sub(r'\.To\(BeEmpty\b', '.To(gomega.BeEmpty', content)
    content = re.sub(r'\.To\(BeNil\b', '.To(gomega.BeNil', content)
    content = re.sub(r'\.NotTo\(BeNil\b', '.NotTo(gomega.BeNil', content)
    content = re.sub(r'\.NotTo\(BeEmpty\b', '.NotTo(gomega.BeEmpty', content)

    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"Fixed dot-imports in {file_path}")

def main():
    """Fix dot-imports in all specified integration test files."""

    base_dir = r"C:\Users\thc1006\Desktop\dev\O-RAN-Intent-MANO-for-Network-Slicing"
    files_to_fix = [
        "tests/integration/vnf_lifecycle_test.go",
        "tests/integration/multi_cluster_test.go",
        "tests/integration/nephio_integration_test.go",
        "tests/integration/o2_interface_test.go"
    ]

    for file_path in files_to_fix:
        full_path = os.path.join(base_dir, file_path)
        if os.path.exists(full_path):
            fix_dot_imports(full_path)
        else:
            print(f"File not found: {full_path}")

if __name__ == "__main__":
    main()