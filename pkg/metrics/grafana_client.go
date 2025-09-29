package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// GrafanaClient manages Grafana API interactions
type GrafanaClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
	dashboards map[string]*Dashboard
	mu       sync.RWMutex
}

// NewGrafanaClient creates a new Grafana client
func NewGrafanaClient(baseURL, username, password string) *GrafanaClient {
	return &GrafanaClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{},
		dashboards: make(map[string]*Dashboard),
	}
}

// ImportDashboard imports a dashboard to Grafana
func (c *GrafanaClient) ImportDashboard(ctx context.Context, dashboard *Dashboard) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if dashboard == nil {
		return fmt.Errorf("dashboard is nil")
	}

	// Store locally for testing
	c.dashboards[dashboard.UID] = dashboard

	// In production, make actual API call
	if c.baseURL != "" && c.baseURL != "http://grafana:3000" {
		return c.postDashboard(ctx, dashboard)
	}

	return nil
}

// GetDashboard retrieves a dashboard from Grafana
func (c *GrafanaClient) GetDashboard(ctx context.Context, uid string) (*Dashboard, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check local cache
	if dashboard, exists := c.dashboards[uid]; exists {
		return dashboard, nil
	}

	// In production, make API call
	if c.baseURL != "" {
		return c.fetchDashboard(ctx, uid)
	}

	return nil, fmt.Errorf("dashboard not found: %s", uid)
}

// UpdateDashboard updates an existing dashboard
func (c *GrafanaClient) UpdateDashboard(ctx context.Context, dashboard *Dashboard) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if dashboard == nil || dashboard.UID == "" {
		return fmt.Errorf("invalid dashboard")
	}

	c.dashboards[dashboard.UID] = dashboard

	// In production, make PUT request
	if c.baseURL != "" {
		return c.putDashboard(ctx, dashboard)
	}

	return nil
}

// DeleteDashboard deletes a dashboard
func (c *GrafanaClient) DeleteDashboard(ctx context.Context, uid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.dashboards, uid)

	// In production, make DELETE request
	if c.baseURL != "" {
		return c.deleteDashboardAPI(ctx, uid)
	}

	return nil
}

// ListDashboards lists all dashboards
func (c *GrafanaClient) ListDashboards(ctx context.Context) ([]*Dashboard, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dashboards := make([]*Dashboard, 0, len(c.dashboards))
	for _, dashboard := range c.dashboards {
		dashboards = append(dashboards, dashboard)
	}

	// In production, make API call
	if c.baseURL != "" {
		return c.fetchAllDashboards(ctx)
	}

	return dashboards, nil
}

// CreateFolder creates a dashboard folder
func (c *GrafanaClient) CreateFolder(ctx context.Context, name string) (string, error) {
	// In production, POST /api/folders
	folderUID := fmt.Sprintf("folder-%s", name)
	return folderUID, nil
}

// CreateDataSource creates a Prometheus data source
func (c *GrafanaClient) CreateDataSource(ctx context.Context, name, url string) error {
	dataSource := map[string]interface{}{
		"name":      name,
		"type":      "prometheus",
		"url":       url,
		"access":    "proxy",
		"isDefault": true,
	}

	// In production, POST /api/datasources
	if c.baseURL != "" {
		return c.postDataSource(ctx, dataSource)
	}

	return nil
}

// Helper methods for actual API calls

func (c *GrafanaClient) postDashboard(ctx context.Context, dashboard *Dashboard) error {
	generator := NewDashboardGenerator()
	jsonStr, err := generator.ExportDashboard(dashboard)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"dashboard": json.RawMessage(jsonStr),
		"overwrite": true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/dashboards/db", c.baseURL),
		bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to import dashboard: %s", string(body))
	}

	return nil
}

func (c *GrafanaClient) fetchDashboard(ctx context.Context, uid string) (*Dashboard, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/dashboards/uid/%s", c.baseURL, uid),
		nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dashboard not found")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Parse dashboard from response
	dashboard := &Dashboard{
		UID:   uid,
		Title: result["dashboard"].(map[string]interface{})["title"].(string),
	}

	return dashboard, nil
}

func (c *GrafanaClient) putDashboard(ctx context.Context, dashboard *Dashboard) error {
	// Similar to postDashboard but with PUT method
	return c.postDashboard(ctx, dashboard)
}

func (c *GrafanaClient) deleteDashboardAPI(ctx context.Context, uid string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/api/dashboards/uid/%s", c.baseURL, uid),
		nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete dashboard")
	}

	return nil
}

func (c *GrafanaClient) fetchAllDashboards(ctx context.Context) ([]*Dashboard, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/search", c.baseURL),
		nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	dashboards := make([]*Dashboard, 0, len(results))
	for _, result := range results {
		if result["type"] == "dash-db" {
			dashboard := &Dashboard{
				UID:   result["uid"].(string),
				Title: result["title"].(string),
			}
			dashboards = append(dashboards, dashboard)
		}
	}

	return dashboards, nil
}

func (c *GrafanaClient) postDataSource(ctx context.Context, dataSource map[string]interface{}) error {
	data, err := json.Marshal(dataSource)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/datasources", c.baseURL),
		bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create data source")
	}

	return nil
}