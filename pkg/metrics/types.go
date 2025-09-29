package metrics

import (
	"time"
)

// MetricQuery represents a metric query
type MetricQuery struct {
	Name      string
	SliceID   string
	Component string
	TimeRange TimeRange
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
	Step  time.Duration
}

// MetricResult represents query results
type MetricResult struct {
	Name   string
	Values []MetricValue
}

// MetricValue represents a single metric value
type MetricValue struct {
	Value     float64
	Timestamp time.Time
}

// CoreMetrics represents 5G Core metrics
type CoreMetrics struct {
	RegisteredUEs  int
	ActiveSessions int
	AMFLoad        float64
	SMFLoad        float64
	UPFThroughput  float64
}

// AggregationQuery represents an aggregation query
type AggregationQuery struct {
	Metric    string
	GroupBy   string
	Operation AggregationOp
	Sites     []string
}

// AggregationOp represents aggregation operations
type AggregationOp string

const (
	AggregationSum AggregationOp = "sum"
	AggregationAvg AggregationOp = "avg"
	AggregationMax AggregationOp = "max"
	AggregationMin AggregationOp = "min"
)

// AggregatedMetrics represents aggregated results
type AggregatedMetrics struct {
	TotalValue  float64
	GroupValues map[string]float64
}

// AlertRule represents a Prometheus alert rule
type AlertRule struct {
	Name        string
	Expression  string
	Duration    time.Duration
	Severity    AlertSeverity
	Annotations map[string]string
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents an active alert
type Alert struct {
	Name  string
	State string
}

// DashboardConfig represents dashboard configuration
type DashboardConfig struct {
	Title       string
	Description string
	Panels      []PanelConfig
}

// PanelConfig represents a dashboard panel configuration
type PanelConfig struct {
	Title string
	Type  PanelType
	Query string
	Unit  string
}

// PanelType represents panel types
type PanelType string

const (
	PanelTypeGraph PanelType = "graph"
	PanelTypeStat  PanelType = "stat"
	PanelTypeTable PanelType = "table"
)

// Dashboard represents a generated dashboard
type Dashboard struct {
	UID    string
	Title  string
	Panels []Panel
}

// Panel represents a dashboard panel
type Panel struct {
	ID    int
	Title string
	Type  PanelType
}

// HistoricalQuery represents a historical data query
type HistoricalQuery struct {
	Metric string
	Start  time.Time
	End    time.Time
	Step   time.Duration
}

// HistoricalData represents historical metric data
type HistoricalData struct {
	DataPoints []DataPoint
}

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// SLAConfig represents SLA configuration
type SLAConfig struct {
	SliceID          string
	LatencyThreshold float64 // ms
	Availability     float64 // percentage
	Period           time.Duration
}

// SLACompliance represents SLA compliance results
type SLACompliance struct {
	LatencyCompliance    float64
	AvailabilityMeasured float64
}

// ExportConfig represents export configuration
type ExportConfig struct {
	Metrics   []string
	Format    ExportFormat
	TimeRange TimeRange
}

// ExportFormat represents export formats
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportData represents exported data
type ExportData struct {
	Content string
	Format  ExportFormat
}

// StreamConfig represents streaming configuration
type StreamConfig struct {
	Metrics  []string
	Interval time.Duration
}

// StreamedMetric represents a streamed metric
type StreamedMetric struct {
	Name      string
	Value     float64
	Timestamp time.Time
}