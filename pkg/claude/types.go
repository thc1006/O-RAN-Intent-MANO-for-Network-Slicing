package claude

import "time"

// ClientConfig represents Claude client configuration
type ClientConfig struct {
	SessionName string
	Timeout     time.Duration
	UseFallback bool
}

// IntentRequest represents a natural language intent request
type IntentRequest struct {
	Text    string
	Context Context
}

// Context represents the context for intent processing
type Context struct {
	Domain  string
	Type    string
	SliceID string
}

// IntentResponse represents the processed intent response
type IntentResponse struct {
	ActionType   string
	ParsedIntent *ParsedIntent
	ParsedSlices []*ParsedSlice
	QoSUpdate    *QoSUpdate
	TargetSlice  string
	Response     string
}

// ParsedIntent represents a parsed network slice intent
type ParsedIntent struct {
	SliceType    string
	Requirements *Requirements
}

// ParsedSlice represents a parsed network slice
type ParsedSlice struct {
	Type         string
	Name         string
	Requirements *Requirements
}

// Requirements represents slice requirements
type Requirements struct {
	Throughput  int     // Mbps
	Latency     float64 // ms
	Reliability float64 // percentage
}

// QoSUpdate represents QoS update parameters
type QoSUpdate struct {
	Latency     float64
	Throughput  int
	Reliability float64
}

// BatchResult represents batch processing result
type BatchResult struct {
	Intent   string
	Response *IntentResponse
	Success  bool
	Error    error
}

// PromptTemplate represents a prompt template
type PromptTemplate struct {
	Type       string
	Parameters map[string]interface{}
}

// QoSParameters represents QoS validation parameters
type QoSParameters struct {
	Latency     float64
	Throughput  int
	Reliability float64
}