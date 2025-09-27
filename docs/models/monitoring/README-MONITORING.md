# O-RAN MANO Monitoring Architecture / O-RAN MANO監控架構

## 監控架構概述 / Monitoring Architecture Overview

本文檔描述了O-RAN MANO系統的完整監控架構，包括指標收集、日誌聚合、分散式追蹤和告警管理。該架構遵循MBSE(Model-Based Systems Engineering)原則，為TDD(Test-Driven Development)實現提供了清晰的測試場景。

This document describes the comprehensive monitoring architecture for the O-RAN MANO system, including metrics collection, log aggregation, distributed tracing, and alert management. The architecture follows MBSE (Model-Based Systems Engineering) principles and provides clear test scenarios for TDD (Test-Driven Development) implementation.

## 核心組件 / Core Components

### 1. Prometheus生態系統 / Prometheus Ecosystem

#### Prometheus Operator
- **功能 / Features**: 聲明式配置管理 / Declarative configuration management
- **CRDs**: ServiceMonitor, PodMonitor, PrometheusRule, AlertManager
- **自動發現 / Auto-discovery**: 基於Kubernetes標籤的服務發現 / Kubernetes label-based service discovery

#### Prometheus Server
- **抓取間隔 / Scrape Interval**: 15秒 (可配置) / 15 seconds (configurable)
- **資料保留 / Data Retention**: 本地30天 / 30 days local
- **高可用性 / High Availability**: 多實例部署 / Multi-instance deployment
- **儲存 / Storage**: TSDB + 長期物件儲存 / TSDB + long-term object storage

#### AlertManager集群 / AlertManager Cluster
- **集群配置 / Cluster Setup**: 3節點高可用 / 3-node high availability
- **去重處理 / Deduplication**: 跨實例告警去重 / Cross-instance alert deduplication
- **路由策略 / Routing**: 基於標籤的告警路由 / Label-based alert routing

### 2. Grafana視覺化平台 / Grafana Visualization Platform

#### 核心功能 / Core Features
- **儀表板 / Dashboards**: 50+預建儀表板 / 50+ pre-built dashboards
- **資料來源 / Data Sources**: Prometheus, Loki, Jaeger, PostgreSQL
- **告警 / Alerting**: 整合AlertManager / Integrated with AlertManager
- **權限控制 / Access Control**: 基於角色的存取控制 / Role-based access control

#### 儀表板類別 / Dashboard Categories
1. **基礎設施監控 / Infrastructure Monitoring**
2. **O-RAN組件監控 / O-RAN Component Monitoring**
3. **應用效能監控 / Application Performance Monitoring**
4. **業務指標監控 / Business Metrics Monitoring**
5. **告警管理 / Alert Management**

### 3. 可觀測性堆疊 / Observability Stack

#### 三大支柱 / Three Pillars
1. **指標 / Metrics**: Prometheus + VictoriaMetrics
2. **日誌 / Logs**: Loki + Fluent Bit
3. **追蹤 / Traces**: Jaeger + Tempo

#### 資料流程 / Data Flow
```
應用程式 → 檢測層 → 收集層 → 儲存層 → 查詢層 → 視覺化層
Application → Instrumentation → Collection → Storage → Query → Visualization
```

## O-RAN組件指標目錄 / O-RAN Component Metrics Catalog

### O2 DMS (O2 Device Management Service)

#### 業務指標 / Business Metrics
```prometheus
# 設備註冊數量 / Device Registration Count
o2dms_device_registrations_total{device_type, status}

# 設備健康狀態 / Device Health Status
o2dms_device_health_status{device_id, status}

# 配置同步成功率 / Configuration Sync Success Rate
o2dms_config_sync_success_rate{device_type}

# 故障管理響應時間 / Fault Management Response Time
o2dms_fault_management_response_duration_seconds{severity}
```

#### 技術指標 / Technical Metrics
```prometheus
# HTTP請求持續時間 / HTTP Request Duration
o2dms_http_request_duration_seconds{method, endpoint, status_code}

# HTTP請求總數 / HTTP Request Total
o2dms_http_requests_total{method, endpoint, status_code}

# 資料庫連接池 / Database Connection Pool
o2dms_db_connections{state="active|idle|waiting"}

# 快取命中率 / Cache Hit Rate
o2dms_cache_hit_rate{cache_type}
```

### CN DMS (Cloud-Native Device Management Service)

#### 業務指標 / Business Metrics
```prometheus
# VNF生命週期管理 / VNF Lifecycle Management
cndms_vnf_lifecycle_operations_total{operation, status}

# 網路服務實例 / Network Service Instances
cndms_network_service_instances{status}

# 資源利用率 / Resource Utilization
cndms_resource_utilization_ratio{resource_type}

# SLA合規性 / SLA Compliance
cndms_sla_compliance_ratio{service_type}
```

#### 技術指標 / Technical Metrics
```prometheus
# gRPC請求持續時間 / gRPC Request Duration
cndms_grpc_request_duration_seconds{method, status_code}

# 容器資源使用 / Container Resource Usage
cndms_container_resource_usage{resource="cpu|memory|disk", container}

# 訊息佇列深度 / Message Queue Depth
cndms_message_queue_depth{queue_name}

# 服務網格指標 / Service Mesh Metrics
cndms_service_mesh_requests_total{source, destination, response_code}
```

### VNF Operator

#### 業務指標 / Business Metrics
```prometheus
# VNF部署成功率 / VNF Deployment Success Rate
vnfoperator_deployment_success_rate{vnf_type}

# VNF擴縮容操作 / VNF Scaling Operations
vnfoperator_scaling_operations_total{vnf_id, operation}

# 自動修復次數 / Auto-healing Count
vnfoperator_auto_healing_total{vnf_id, reason}

# 配置漂移檢測 / Configuration Drift Detection
vnfoperator_config_drift_detected_total{vnf_id}
```

#### 技術指標 / Technical Metrics
```prometheus
# Kubernetes API調用 / Kubernetes API Calls
vnfoperator_k8s_api_calls_total{operation, resource, status}

# 控制器調和時間 / Controller Reconcile Time
vnfoperator_controller_reconcile_duration_seconds{controller}

# 工作佇列深度 / Work Queue Depth
vnfoperator_workqueue_depth{name}

# 領導者選舉 / Leader Election
vnfoperator_leader_election_status{instance}
```

### Intent Service

#### 業務指標 / Business Metrics
```prometheus
# 意圖處理延遲 / Intent Processing Latency
intent_service_processing_duration_seconds{intent_type}

# 意圖成功率 / Intent Success Rate
intent_service_success_rate{intent_type}

# 網路切片創建 / Network Slice Creation
intent_service_network_slice_operations_total{operation, status}

# QoS合規性 / QoS Compliance
intent_service_qos_compliance_ratio{slice_id}
```

#### 技術指標 / Technical Metrics
```prometheus
# REST API效能 / REST API Performance
intent_service_api_request_duration_seconds{endpoint, method}

# 策略引擎效能 / Policy Engine Performance
intent_service_policy_evaluation_duration_seconds{policy_type}

# 事件處理 / Event Processing
intent_service_events_processed_total{event_type, status}

# 外部API調用 / External API Calls
intent_service_external_api_calls_total{service, operation, status}
```

## 告警定義和嚴重程度 / Alert Definitions and Severity Levels

### 嚴重程度分級 / Severity Classification

#### Critical (嚴重)
- **服務完全不可用 / Service Completely Unavailable**
- **資料遺失風險 / Data Loss Risk**
- **安全漏洞 / Security Vulnerabilities**
- **SLA違反 / SLA Violations**

#### Warning (警告)
- **效能下降 / Performance Degradation**
- **資源使用率高 / High Resource Utilization**
- **間歇性錯誤 / Intermittent Errors**
- **容量預警 / Capacity Warnings**

#### Info (資訊)
- **配置變更 / Configuration Changes**
- **部署事件 / Deployment Events**
- **維護通知 / Maintenance Notifications**
- **統計資訊 / Statistical Information**

### 告警規則範例 / Alert Rule Examples

#### 高CPU使用率 / High CPU Usage
```yaml
alert: HighCPUUsage
expr: (100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)) > 90
for: 5m
labels:
  severity: critical
annotations:
  summary: "High CPU usage detected"
  description: "CPU usage is above 90% for more than 5 minutes"
```

#### O2 DMS服務下線 / O2 DMS Service Down
```yaml
alert: O2DMSServiceDown
expr: up{job="o2dms"} == 0
for: 1m
labels:
  severity: critical
  service: o2dms
annotations:
  summary: "O2 DMS service is down"
  description: "O2 DMS service has been down for more than 1 minute"
```

#### 意圖處理延遲高 / High Intent Processing Latency
```yaml
alert: HighIntentProcessingLatency
expr: histogram_quantile(0.95, rate(intent_service_processing_duration_seconds_bucket[5m])) > 30
for: 5m
labels:
  severity: warning
  service: intent-service
annotations:
  summary: "High intent processing latency"
  description: "95th percentile latency is above 30 seconds"
```

## 儀表板描述 / Dashboard Descriptions

### 1. Kubernetes總覽儀表板 / Kubernetes Overview Dashboard
- **集群健康狀態 / Cluster Health Status**
- **節點資源使用 / Node Resource Usage**
- **Pod狀態分佈 / Pod Status Distribution**
- **網路流量統計 / Network Traffic Statistics**

### 2. O-RAN組件儀表板 / O-RAN Component Dashboard
- **組件可用性 / Component Availability**
- **API回應時間 / API Response Times**
- **錯誤率趨勢 / Error Rate Trends**
- **吞吐量監控 / Throughput Monitoring**

### 3. 業務指標儀表板 / Business Metrics Dashboard
- **意圖處理統計 / Intent Processing Statistics**
- **網路切片KPI / Network Slice KPIs**
- **QoS合規性報告 / QoS Compliance Reports**
- **SLA達成率 / SLA Achievement Rate**

### 4. 基礎設施監控儀表板 / Infrastructure Monitoring Dashboard
- **系統資源監控 / System Resource Monitoring**
- **網路效能 / Network Performance**
- **儲存使用情況 / Storage Utilization**
- **安全事件 / Security Events**

## TDD測試場景推導 / TDD Test Scenario Derivation

### 從MBSE模型推導測試場景 / Deriving Test Scenarios from MBSE Models

#### 1. 部署架構測試 / Deployment Architecture Tests
```go
func TestKubernetesDeploymentArchitecture(t *testing.T) {
    // 測試集群連接性 / Test cluster connectivity
    // 測試命名空間創建 / Test namespace creation
    // 測試資源配額 / Test resource quotas
    // 測試網路策略 / Test network policies
}

func TestPrometheusOperatorDeployment(t *testing.T) {
    // 測試Operator安裝 / Test Operator installation
    // 測試CRD創建 / Test CRD creation
    // 測試ServiceMonitor配置 / Test ServiceMonitor configuration
    // 測試告警規則 / Test alert rules
}
```

#### 2. 指標收集測試 / Metrics Collection Tests
```go
func TestMetricsExposure(t *testing.T) {
    // 測試/metrics端點可用性 / Test /metrics endpoint availability
    // 測試指標格式正確性 / Test metrics format correctness
    // 測試標籤一致性 / Test label consistency
    // 測試指標值合理性 / Test metric value reasonableness
}

func TestPrometheusScrapingE2E(t *testing.T) {
    // 測試服務發現 / Test service discovery
    // 測試指標抓取 / Test metrics scraping
    // 測試資料儲存 / Test data storage
    // 測試查詢功能 / Test query functionality
}
```

#### 3. 告警系統測試 / Alerting System Tests
```go
func TestAlertManagerCluster(t *testing.T) {
    // 測試集群同步 / Test cluster synchronization
    // 測試告警去重 / Test alert deduplication
    // 測試路由規則 / Test routing rules
    // 測試通知發送 / Test notification delivery
}

func TestAlertEscalation(t *testing.T) {
    // 測試升級時序 / Test escalation timing
    // 測試多通道通知 / Test multi-channel notifications
    // 測試確認機制 / Test acknowledgment mechanism
    // 測試自動修復 / Test auto-remediation
}
```

#### 4. 儀表板測試 / Dashboard Tests
```go
func TestGrafanaDashboards(t *testing.T) {
    // 測試儀表板載入 / Test dashboard loading
    // 測試資料來源連接 / Test data source connections
    // 測試面板渲染 / Test panel rendering
    // 測試告警設定 / Test alert configuration
}

func TestDashboardDataAccuracy(t *testing.T) {
    // 測試資料準確性 / Test data accuracy
    // 測試時間範圍選擇 / Test time range selection
    // 測試查詢效能 / Test query performance
    // 測試使用者權限 / Test user permissions
}
```

### 整合測試場景 / Integration Test Scenarios

#### 端到端監控測試 / End-to-End Monitoring Test
```go
func TestE2EMonitoringWorkflow(t *testing.T) {
    // 1. 部署應用程式 / Deploy application
    // 2. 配置監控 / Configure monitoring
    // 3. 產生測試負載 / Generate test load
    // 4. 驗證指標收集 / Verify metrics collection
    // 5. 觸發告警 / Trigger alerts
    // 6. 驗證告警傳播 / Verify alert propagation
    // 7. 檢查儀表板更新 / Check dashboard updates
    // 8. 驗證資料一致性 / Verify data consistency
}
```

#### 災難恢復測試 / Disaster Recovery Test
```go
func TestMonitoringDisasterRecovery(t *testing.T) {
    // 測試組件故障處理 / Test component failure handling
    // 測試資料備份恢復 / Test data backup and recovery
    // 測試高可用性切換 / Test high availability switching
    // 測試系統自愈能力 / Test system self-healing capabilities
}
```

## 故障排除指南 / Troubleshooting Guide

### 常見問題 / Common Issues

#### 1. Prometheus抓取失敗 / Prometheus Scraping Failures
**症狀 / Symptoms**:
- 目標顯示為DOWN狀態 / Targets showing as DOWN
- 指標資料缺失 / Missing metrics data

**診斷步驟 / Diagnostic Steps**:
```bash
# 檢查ServiceMonitor配置 / Check ServiceMonitor configuration
kubectl get servicemonitor -n monitoring

# 檢查端點可達性 / Check endpoint reachability
kubectl get endpoints -n oran-system

# 檢查網路策略 / Check network policies
kubectl get networkpolicy -n oran-system

# 檢查Prometheus配置 / Check Prometheus configuration
kubectl get prometheus -o yaml
```

#### 2. 告警未觸發 / Alerts Not Triggering
**症狀 / Symptoms**:
- 期望的告警未產生 / Expected alerts not generated
- 告警延遲 / Alert delays

**診斷步驟 / Diagnostic Steps**:
```bash
# 檢查告警規則 / Check alert rules
kubectl get prometheusrule -n monitoring

# 檢查AlertManager狀態 / Check AlertManager status
kubectl get pods -n monitoring -l app=alertmanager

# 驗證告警規則語法 / Verify alert rule syntax
promtool check rules /etc/prometheus/rules/*.yml
```

#### 3. 儀表板載入緩慢 / Slow Dashboard Loading
**症狀 / Symptoms**:
- 儀表板載入時間長 / Long dashboard loading times
- 查詢超時 / Query timeouts

**優化建議 / Optimization Recommendations**:
- 使用記錄規則預計算複雜查詢 / Use recording rules for complex queries
- 限制時間範圍和資料點數量 / Limit time range and data points
- 優化PromQL查詢 / Optimize PromQL queries
- 增加Prometheus記憶體 / Increase Prometheus memory

### 效能調優 / Performance Tuning

#### Prometheus調優 / Prometheus Tuning
```yaml
# prometheus.yml配置建議 / Prometheus.yml configuration recommendations
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'oran-mano'

rule_files:
  - "rules/*.yml"

storage:
  tsdb:
    retention.time: 30d
    retention.size: 100GB
    wal-compression: true
```

#### Grafana調優 / Grafana Tuning
```ini
# grafana.ini配置建議 / Grafana.ini configuration recommendations
[database]
type = postgres
max_open_conn = 100
max_idle_conn = 100

[caching]
enabled = true

[query_cache]
enabled = true
ttl = 5m
max_cache_size_mb = 100
```

## 安全考慮 / Security Considerations

### 存取控制 / Access Control
- **RBAC配置 / RBAC Configuration**: 基於最小權限原則 / Based on principle of least privilege
- **API金鑰管理 / API Key Management**: 定期輪換和權限限制 / Regular rotation and permission limits
- **網路分隔 / Network Segmentation**: 監控組件網路隔離 / Monitor component network isolation

### 資料保護 / Data Protection
- **加密傳輸 / Encrypted Transit**: 所有資料傳輸使用TLS / All data transit uses TLS
- **資料脫敏 / Data Masking**: 敏感資料自動脫敏 / Automatic masking of sensitive data
- **備份安全 / Backup Security**: 加密備份資料 / Encrypted backup data

## 容量規劃 / Capacity Planning

### 資源需求估算 / Resource Requirements Estimation

#### Prometheus
- **CPU**: 每1000個目標2核心 / 2 cores per 1000 targets
- **記憶體**: 每100萬個樣本2GB / 2GB per 1M samples
- **儲存**: 每100萬個樣本1.3字節 / 1.3 bytes per 1M samples

#### Grafana
- **CPU**: 基礎2核心 / Base 2 cores
- **記憶體**: 4GB起始 / 4GB minimum
- **儲存**: 10GB配置資料 / 10GB for configuration

#### AlertManager
- **CPU**: 1核心 / 1 core
- **記憶體**: 512MB / 512MB
- **儲存**: 1GB告警歷史 / 1GB for alert history

### 擴展策略 / Scaling Strategy
- **水平擴展 / Horizontal Scaling**: 多實例部署 / Multi-instance deployment
- **聯邦架構 / Federation Architecture**: 跨集群監控 / Cross-cluster monitoring
- **資料分層 / Data Tiering**: 熱溫冷資料分離 / Hot-warm-cold data separation

---

**文件版本 / Document Version**: v1.0
**最後更新 / Last Updated**: 2024-09-27
**作者 / Author**: O-RAN MANO Architecture Team
**審核 / Reviewed**: System Architecture Team

本文檔遵循MBSE原則，為O-RAN MANO監控系統提供了完整的架構設計和實現指導，同時支援TDD開發方法論。

This document follows MBSE principles and provides comprehensive architectural design and implementation guidance for the O-RAN MANO monitoring system, while supporting TDD development methodology.