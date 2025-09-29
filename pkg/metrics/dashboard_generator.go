package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// DashboardGenerator generates Grafana dashboards
type DashboardGenerator struct {
	templates map[string]*DashboardTemplate
}

// DashboardTemplate represents a dashboard template
type DashboardTemplate struct {
	Name        string
	Description string
	Panels      []PanelTemplate
}

// PanelTemplate represents a panel template
type PanelTemplate struct {
	Title       string
	Type        PanelType
	Query       string
	Unit        string
	GridPos     GridPosition
}

// GridPosition represents panel position on dashboard
type GridPosition struct {
	X      int
	Y      int
	Width  int
	Height int
}

// NewDashboardGenerator creates a new dashboard generator
func NewDashboardGenerator() *DashboardGenerator {
	generator := &DashboardGenerator{
		templates: make(map[string]*DashboardTemplate),
	}

	// Initialize default templates
	generator.initializeTemplates()
	return generator
}

// GenerateDashboard generates a dashboard from config
func (g *DashboardGenerator) GenerateDashboard(ctx context.Context, config *DashboardConfig) (*Dashboard, error) {
	if config == nil || config.Title == "" {
		return nil, fmt.Errorf("invalid dashboard config")
	}

	dashboard := &Dashboard{
		UID:    uuid.New().String()[:8],
		Title:  config.Title,
		Panels: []Panel{},
	}

	// Create panels from config
	for i, panelConfig := range config.Panels {
		panel := Panel{
			ID:    i + 1,
			Title: panelConfig.Title,
			Type:  panelConfig.Type,
		}
		dashboard.Panels = append(dashboard.Panels, panel)
	}

	return dashboard, nil
}

// GenerateRANDashboard generates RAN-specific dashboard
func (g *DashboardGenerator) GenerateRANDashboard(ctx context.Context, siteID string) (*Dashboard, error) {
	dashboard := &Dashboard{
		UID:    fmt.Sprintf("ran-%s", siteID),
		Title:  fmt.Sprintf("RAN Components - %s", siteID),
		Panels: []Panel{},
	}

	// Add standard RAN panels
	ranPanels := []struct {
		title     string
		panelType PanelType
	}{
		{"CU-CP Sessions", PanelTypeGraph},
		{"DU PRB Utilization", PanelTypeGraph},
		{"RU Power Consumption", PanelTypeGraph},
		{"Handover Success Rate", PanelTypeStat},
		{"Connected UEs", PanelTypeStat},
		{"Throughput", PanelTypeGraph},
		{"Latency", PanelTypeGraph},
		{"Packet Loss", PanelTypeStat},
	}

	for i, p := range ranPanels {
		panel := Panel{
			ID:    i + 1,
			Title: p.title,
			Type:  p.panelType,
		}
		dashboard.Panels = append(dashboard.Panels, panel)
	}

	return dashboard, nil
}

// GenerateSliceDashboard generates network slice dashboard
func (g *DashboardGenerator) GenerateSliceDashboard(ctx context.Context, sliceType string) (*Dashboard, error) {
	dashboard := &Dashboard{
		UID:    fmt.Sprintf("slice-%s", sliceType),
		Title:  fmt.Sprintf("Network Slice - %s", sliceType),
		Panels: []Panel{},
	}

	// Add slice-specific panels based on type
	switch sliceType {
	case "eMBB":
		dashboard.Panels = g.generateEMBBPanels()
	case "URLLC":
		dashboard.Panels = g.generateURLLCPanels()
	case "mIoT":
		dashboard.Panels = g.generateMIoTPanels()
	default:
		dashboard.Panels = g.generateDefaultPanels()
	}

	return dashboard, nil
}

// ExportDashboard exports dashboard to JSON
func (g *DashboardGenerator) ExportDashboard(dashboard *Dashboard) (string, error) {
	if dashboard == nil {
		return "", fmt.Errorf("dashboard is nil")
	}

	// Create Grafana-compatible JSON
	grafanaJSON := g.createGrafanaJSON(dashboard)

	data, err := json.MarshalIndent(grafanaJSON, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Helper methods

func (g *DashboardGenerator) initializeTemplates() {
	// Initialize common templates
	g.templates["network-slice"] = &DashboardTemplate{
		Name:        "Network Slice Performance",
		Description: "Monitor network slice KPIs",
		Panels: []PanelTemplate{
			{
				Title: "Throughput",
				Type:  PanelTypeGraph,
				Query: "rate(network_slice_throughput[5m])",
				Unit:  "Mbps",
				GridPos: GridPosition{X: 0, Y: 0, Width: 12, Height: 8},
			},
			{
				Title: "Latency",
				Type:  PanelTypeGraph,
				Query: "network_slice_latency",
				Unit:  "ms",
				GridPos: GridPosition{X: 12, Y: 0, Width: 12, Height: 8},
			},
		},
	}
}

func (g *DashboardGenerator) generateEMBBPanels() []Panel {
	return []Panel{
		{ID: 1, Title: "Throughput (Gbps)", Type: PanelTypeGraph},
		{ID: 2, Title: "Active Sessions", Type: PanelTypeStat},
		{ID: 3, Title: "Video Streaming Quality", Type: PanelTypeGraph},
		{ID: 4, Title: "Buffer Health", Type: PanelTypeStat},
	}
}

func (g *DashboardGenerator) generateURLLCPanels() []Panel {
	return []Panel{
		{ID: 1, Title: "Latency (μs)", Type: PanelTypeGraph},
		{ID: 2, Title: "Reliability (%)", Type: PanelTypeStat},
		{ID: 3, Title: "Jitter", Type: PanelTypeGraph},
		{ID: 4, Title: "Packet Loss Rate", Type: PanelTypeStat},
	}
}

func (g *DashboardGenerator) generateMIoTPanels() []Panel {
	return []Panel{
		{ID: 1, Title: "Connected Devices", Type: PanelTypeStat},
		{ID: 2, Title: "Message Rate", Type: PanelTypeGraph},
		{ID: 3, Title: "Battery Efficiency", Type: PanelTypeGraph},
		{ID: 4, Title: "Coverage Area", Type: PanelTypeStat},
	}
}

func (g *DashboardGenerator) generateDefaultPanels() []Panel {
	return []Panel{
		{ID: 1, Title: "Throughput", Type: PanelTypeGraph},
		{ID: 2, Title: "Latency", Type: PanelTypeGraph},
		{ID: 3, Title: "Active UEs", Type: PanelTypeStat},
		{ID: 4, Title: "Resource Utilization", Type: PanelTypeGraph},
	}
}

func (g *DashboardGenerator) createGrafanaJSON(dashboard *Dashboard) map[string]interface{} {
	panels := []map[string]interface{}{}

	for i, panel := range dashboard.Panels {
		panelJSON := map[string]interface{}{
			"id":       panel.ID,
			"title":    panel.Title,
			"type":     string(panel.Type),
			"gridPos": map[string]int{
				"x":      (i % 2) * 12,
				"y":      (i / 2) * 8,
				"w":      12,
				"h":      8,
			},
			"targets": []map[string]interface{}{
				{
					"refId": "A",
					"expr":  fmt.Sprintf("metric_%d", panel.ID),
				},
			},
		}
		panels = append(panels, panelJSON)
	}

	return map[string]interface{}{
		"uid":         dashboard.UID,
		"title":       dashboard.Title,
		"panels":      panels,
		"schemaVersion": 27,
		"version":     1,
		"timezone":    "browser",
		"refresh":     "5s",
		"time": map[string]string{
			"from": "now-6h",
			"to":   "now",
		},
	}
}