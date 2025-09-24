#!/bin/bash

# Fix static analysis issues script

echo "🔧 Fixing static analysis issues..."

# Fix 1: unparam - Remove unused error returns in argocd_validator.go
echo "Fixing argocd_validator.go..."
sed -i 's/func (acv \*ArgoCDValidator) validateApplications(ctx context.Context) (\[\]ApplicationStatus, error)/func (acv *ArgoCDValidator) validateApplications(ctx context.Context) []ApplicationStatus/' clusters/validation-framework/argocd_validator.go
sed -i 's/return applications, nil/return applications/' clusters/validation-framework/argocd_validator.go
sed -i 's/func (acv \*ArgoCDValidator) parseApplicationStatus(app \*unstructured.Unstructured) (\*ApplicationStatus, error)/func (acv *ArgoCDValidator) parseApplicationStatus(app *unstructured.Unstructured) *ApplicationStatus/' clusters/validation-framework/argocd_validator.go

# Fix 2: unparam - Remove unused error returns in metrics_collector.go
echo "Fixing metrics_collector.go..."
sed -i 's/func (mc \*MetricsCollector) CollectMetrics(ctx context.Context) (\[\]ClusterMetrics, error)/func (mc *MetricsCollector) CollectMetrics(ctx context.Context) []ClusterMetrics/' clusters/validation-framework/metrics_collector.go
sed -i 's/return allMetrics, nil/return allMetrics/' clusters/validation-framework/metrics_collector.go

# Fix 3: unparam - Remove unused error returns in rollback_manager.go
echo "Fixing rollback_manager.go..."
sed -i 's/func (rm \*RollbackManager) GetRollbackHistory(ctx context.Context, clusterName string) (\[\]RollbackEvent, error)/func (rm *RollbackManager) GetRollbackHistory(ctx context.Context, clusterName string) []RollbackEvent/' clusters/validation-framework/rollback_manager.go
sed -i 's/return events, nil/return events/' clusters/validation-framework/rollback_manager.go

# Fix 4: unparam - Remove unused error returns in dashboard.go
echo "Fixing dashboard.go..."
sed -i 's/func (d \*Dashboard) GenerateDashboardMetrics() (DashboardMetrics, error)/func (d *Dashboard) GenerateDashboardMetrics() DashboardMetrics/' tests/framework/dashboard/dashboard.go
sed -i 's/return metrics, nil/return metrics/' tests/framework/dashboard/dashboard.go

# Fix 5: unparam - Remove unused error returns in metrics_aggregator.go
echo "Fixing metrics_aggregator.go..."
sed -i 's/func (ma \*MetricsAggregator) AggregateMetrics() (AggregatedMetrics, error)/func (ma *MetricsAggregator) AggregateMetrics() AggregatedMetrics/' tests/framework/dashboard/metrics_aggregator.go
sed -i 's/return aggregated, nil/return aggregated/' tests/framework/dashboard/metrics_aggregator.go

# Fix 6: unparam - Remove unused error returns in e2e_workflow_test.go
echo "Fixing e2e_workflow_test.go..."
sed -i 's/func runE2EWorkflow(ctx context.Context, config WorkflowConfig) (WorkflowResult, error)/func runE2EWorkflow(ctx context.Context, config WorkflowConfig) WorkflowResult/' tests/integration/e2e_workflow_test.go
sed -i 's/return result, nil/return result/' tests/integration/e2e_workflow_test.go

# Fix 7: errcheck - Check os.Setenv error
echo "Fixing git_repository.go..."
sed -i '/os.Setenv("GIT_SSH_COMMAND", sshCmd)/c\
\tif err := os.Setenv("GIT_SSH_COMMAND", sshCmd); err != nil {\
\t\treturn err\
\t}' clusters/validation-framework/git_repository.go

echo "✅ Static analysis fixes applied!"
echo "Note: You may need to manually review and fix:"
echo "  - Unused struct fields in tests/e2e/deployment_timing_test.go"
echo "  - Unused struct fields in tests/e2e/intent_to_slice_workflow_test.go"
echo "  - Update function call sites after removing error returns"