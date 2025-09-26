package intent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IntentParser parses natural language intents into structured QoS profiles
type IntentParser struct {
	logger   *slog.Logger
	patterns map[string]*IntentPattern
}

// Intent represents a parsed user intent
type Intent struct {
	ID           string           `json:"id"`
	RawText      string           `json:"raw_text"`
	ServiceType  ServiceType      `json:"service_type"`
	QoSProfile   *QoSProfile      `json:"qos_profile"`
	SliceConfig  *SliceConfig     `json:"slice_config"`
	Constraints  []Constraint     `json:"constraints"`
	Priority     Priority         `json:"priority"`
	ParsedAt     time.Time        `json:"parsed_at"`
	Confidence   float64          `json:"confidence"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceType represents different types of 5G services
type ServiceType string

const (
	// eMBB - Enhanced Mobile Broadband
	ServiceTypeEMBB ServiceType = "eMBB"
	// URLLC - Ultra-Reliable Low-Latency Communication
	ServiceTypeURLLC ServiceType = "URLLC"
	// mMTC - Massive Machine Type Communication
	ServiceTypeMmTC ServiceType = "mMTC"
	// Custom service type
	ServiceTypeCustom ServiceType = "custom"
)

// QoSProfile defines Quality of Service requirements
type QoSProfile struct {
	// Bandwidth requirements in Mbps
	Bandwidth     QoSRequirement  `json:"bandwidth"`
	// Latency requirements in milliseconds
	Latency       QoSRequirement  `json:"latency"`
	// Jitter requirements in milliseconds
	Jitter        QoSRequirement  `json:"jitter"`
	// Packet loss rate (0-1)
	PacketLoss    QoSRequirement  `json:"packet_loss"`
	// Reliability percentage
	Reliability   QoSRequirement  `json:"reliability"`
	// Throughput requirements in Mbps
	Throughput    QoSRequirement  `json:"throughput"`
}

// QoSRequirement represents a QoS parameter with constraints
type QoSRequirement struct {
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Target   *float64 `json:"target,omitempty"`
	Unit     string   `json:"unit"`
	Critical bool     `json:"critical"`
}

// SliceConfig represents network slice configuration
type SliceConfig struct {
	Name         string            `json:"name"`
	Type         ServiceType       `json:"type"`
	Coverage     Coverage          `json:"coverage"`
	UserDensity  *int              `json:"user_density,omitempty"`
	MobilityLevel MobilityLevel    `json:"mobility_level"`
	DeviceTypes  []string          `json:"device_types"`
	Applications []ApplicationSpec `json:"applications"`
}

// Coverage represents geographic coverage requirements
type Coverage struct {
	Areas      []string  `json:"areas"`
	Radius     *float64  `json:"radius_km,omitempty"`
	Indoor     bool      `json:"indoor"`
	Outdoor    bool      `json:"outdoor"`
	Density    string    `json:"density"` // urban, suburban, rural
}

// MobilityLevel represents user mobility patterns
type MobilityLevel string

const (
	MobilityStationary MobilityLevel = "stationary"
	MobilityPedestrian MobilityLevel = "pedestrian"
	MobilityVehicular  MobilityLevel = "vehicular"
	MobilityAerial     MobilityLevel = "aerial"
)

// ApplicationSpec represents application requirements
type ApplicationSpec struct {
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	QoSClass    int                `json:"qos_class"`
	TrafficPattern TrafficPattern  `json:"traffic_pattern"`
}

// TrafficPattern represents traffic characteristics
type TrafficPattern struct {
	Type         string   `json:"type"` // periodic, burst, constant
	PeakHours    []string `json:"peak_hours,omitempty"`
	BurstFactor  *float64 `json:"burst_factor,omitempty"`
	Predictable  bool     `json:"predictable"`
}

// Constraint represents deployment or operational constraints
type Constraint struct {
	Type        ConstraintType    `json:"type"`
	Value       string            `json:"value"`
	Operator    string            `json:"operator"` // eq, ne, lt, le, gt, ge
	Description string            `json:"description"`
	Mandatory   bool              `json:"mandatory"`
}

// ConstraintType represents different types of constraints
type ConstraintType string

const (
	ConstraintLocation   ConstraintType = "location"
	ConstraintCost       ConstraintType = "cost"
	ConstraintLatency    ConstraintType = "latency"
	ConstraintBandwidth  ConstraintType = "bandwidth"
	ConstraintSecurity   ConstraintType = "security"
	ConstraintCompliance ConstraintType = "compliance"
	ConstraintAvailability ConstraintType = "availability"
)

// Priority represents intent priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// IntentPattern represents a pattern for matching intents
type IntentPattern struct {
	Name        string
	Regex       *regexp.Regexp
	ServiceType ServiceType
	QoSTemplate *QoSProfile
	Confidence  float64
	Extractors  map[string]*regexp.Regexp
}

// NewIntentParser creates a new intent parser with predefined patterns
func NewIntentParser() *IntentParser {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	parser := &IntentParser{
		logger:   logger,
		patterns: make(map[string]*IntentPattern),
	}

	parser.initializePatterns()
	return parser
}

// ParseIntent parses a natural language intent into structured format
func (ip *IntentParser) ParseIntent(ctx context.Context, text string) (*Intent, error) {
	ip.logger.Info("Parsing intent", "text", text)

	intent := &Intent{
		ID:       generateIntentID(),
		RawText:  strings.TrimSpace(text),
		ParsedAt: time.Now(),
		Metadata: make(map[string]interface{}),
	}

	// Normalize input text
	normalizedText := ip.normalizeText(text)

	// Find best matching pattern
	bestMatch, confidence := ip.findBestMatch(normalizedText)
	if bestMatch == nil {
		return nil, fmt.Errorf("no matching pattern found for intent: %s", text)
	}

	intent.ServiceType = bestMatch.ServiceType
	intent.Confidence = confidence

	// Extract QoS requirements
	qosProfile, err := ip.extractQoSProfile(normalizedText, bestMatch)
	if err != nil {
		return nil, fmt.Errorf("failed to extract QoS profile: %w", err)
	}
	intent.QoSProfile = qosProfile

	// Extract constraints
	constraints, err := ip.extractConstraints(normalizedText)
	if err != nil {
		return nil, fmt.Errorf("failed to extract constraints: %w", err)
	}
	intent.Constraints = constraints

	// Extract priority
	priority := ip.extractPriority(normalizedText)
	intent.Priority = priority

	// Generate slice configuration
	sliceConfig, err := ip.generateSliceConfig(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate slice config: %w", err)
	}
	intent.SliceConfig = sliceConfig

	ip.logger.Info("Intent parsed successfully",
		"service_type", intent.ServiceType,
		"confidence", intent.Confidence,
		"priority", intent.Priority)

	return intent, nil
}

// ValidateIntent validates the parsed intent for consistency and completeness
func (ip *IntentParser) ValidateIntent(ctx context.Context, intent *Intent) error {
	if intent == nil {
		return fmt.Errorf("intent is nil")
	}

	if intent.RawText == "" {
		return fmt.Errorf("intent raw text is empty")
	}

	if intent.ServiceType == "" {
		return fmt.Errorf("service type is required")
	}

	if intent.QoSProfile == nil {
		return fmt.Errorf("QoS profile is required")
	}

	// Validate QoS requirements based on service type
	if err := ip.validateQoSProfile(intent.ServiceType, intent.QoSProfile); err != nil {
		return fmt.Errorf("QoS profile validation failed: %w", err)
	}

	// Validate constraints
	for i, constraint := range intent.Constraints {
		if constraint.Type == "" || constraint.Value == "" {
			return fmt.Errorf("constraint %d is incomplete", i)
		}
	}

	ip.logger.Info("Intent validation successful", "intent_id", intent.ID)
	return nil
}

// initializePatterns sets up predefined intent patterns
func (ip *IntentParser) initializePatterns() {
	// Emergency services (URLLC)
	ip.patterns["emergency"] = &IntentPattern{
		Name:        "Emergency Services",
		Regex:       regexp.MustCompile(`(?i)(emergency|critical|urgent|ambulance|fire|police|911|first responder)`),
		ServiceType: ServiceTypeURLLC,
		Confidence:  0.95,
		QoSTemplate: &QoSProfile{
			Latency:     QoSRequirement{Max: floatPtr(1.0), Unit: "ms", Critical: true},
			Bandwidth:   QoSRequirement{Min: floatPtr(4.0), Unit: "Mbps", Critical: true},
			Reliability: QoSRequirement{Min: floatPtr(99.999), Unit: "%", Critical: true},
			PacketLoss:  QoSRequirement{Max: floatPtr(0.001), Unit: "%", Critical: true},
		},
		Extractors: map[string]*regexp.Regexp{
			"latency":   regexp.MustCompile(`(?i)latency[:\s]*([0-9.]+)\s*(ms|milliseconds?)`),
			"bandwidth": regexp.MustCompile(`(?i)bandwidth[:\s]*([0-9.]+)\s*(mbps|gbps)`),
		},
	}

	// Video streaming (eMBB)
	ip.patterns["video"] = &IntentPattern{
		Name:        "Video Streaming",
		Regex:       regexp.MustCompile(`(?i)(video|streaming|media|youtube|netflix|4k|8k|broadcast)`),
		ServiceType: ServiceTypeEMBB,
		Confidence:  0.90,
		QoSTemplate: &QoSProfile{
			Bandwidth:   QoSRequirement{Min: floatPtr(2.5), Target: floatPtr(10.0), Unit: "Mbps", Critical: true},
			Latency:     QoSRequirement{Max: floatPtr(20.0), Unit: "ms", Critical: false},
			Jitter:      QoSRequirement{Max: floatPtr(5.0), Unit: "ms", Critical: true},
			PacketLoss:  QoSRequirement{Max: floatPtr(0.1), Unit: "%", Critical: true},
		},
		Extractors: map[string]*regexp.Regexp{
			"quality":   regexp.MustCompile(`(?i)(4k|8k|hd|uhd|1080p|720p)`),
			"bandwidth": regexp.MustCompile(`(?i)bandwidth[:\s]*([0-9.]+)\s*(mbps|gbps)`),
		},
	}

	// IoT sensors (mMTC)
	ip.patterns["iot"] = &IntentPattern{
		Name:        "IoT Services",
		Regex:       regexp.MustCompile(`(?i)(iot|sensor|smart city|agriculture|monitoring|meters?|device)`),
		ServiceType: ServiceTypeMmTC,
		Confidence:  0.85,
		QoSTemplate: &QoSProfile{
			Bandwidth:  QoSRequirement{Min: floatPtr(0.1), Max: floatPtr(1.0), Unit: "Mbps", Critical: false},
			Latency:    QoSRequirement{Max: floatPtr(100.0), Unit: "ms", Critical: false},
			Reliability: QoSRequirement{Min: floatPtr(99.0), Unit: "%", Critical: true},
		},
		Extractors: map[string]*regexp.Regexp{
			"devices": regexp.MustCompile(`(?i)([0-9,]+)\s*(devices?|sensors?)`),
			"area":    regexp.MustCompile(`(?i)(urban|rural|suburban|indoor|outdoor)`),
		},
	}

	// Autonomous vehicles (URLLC)
	ip.patterns["autonomous"] = &IntentPattern{
		Name:        "Autonomous Vehicles",
		Regex:       regexp.MustCompile(`(?i)(autonomous|self.driving|v2x|vehicle|car|truck|drone|uav)`),
		ServiceType: ServiceTypeURLLC,
		Confidence:  0.93,
		QoSTemplate: &QoSProfile{
			Latency:     QoSRequirement{Max: floatPtr(5.0), Unit: "ms", Critical: true},
			Bandwidth:   QoSRequirement{Min: floatPtr(1.0), Target: floatPtr(5.0), Unit: "Mbps", Critical: true},
			Reliability: QoSRequirement{Min: floatPtr(99.99), Unit: "%", Critical: true},
		},
		Extractors: map[string]*regexp.Regexp{
			"vehicles": regexp.MustCompile(`(?i)([0-9,]+)\s*(vehicles?|cars?|trucks?)`),
			"speed":    regexp.MustCompile(`(?i)([0-9]+)\s*(mph|kmh|km/h)`),
		},
	}

	// Industrial automation (URLLC)
	ip.patterns["industrial"] = &IntentPattern{
		Name:        "Industrial Automation",
		Regex:       regexp.MustCompile(`(?i)(industrial|factory|manufacturing|automation|robotics|plc)`),
		ServiceType: ServiceTypeURLLC,
		Confidence:  0.88,
		QoSTemplate: &QoSProfile{
			Latency:     QoSRequirement{Max: floatPtr(10.0), Unit: "ms", Critical: true},
			Bandwidth:   QoSRequirement{Min: floatPtr(0.5), Target: floatPtr(2.0), Unit: "Mbps", Critical: true},
			Reliability: QoSRequirement{Min: floatPtr(99.9), Unit: "%", Critical: true},
			Jitter:      QoSRequirement{Max: floatPtr(1.0), Unit: "ms", Critical: true},
		},
		Extractors: map[string]*regexp.Regexp{
			"machines": regexp.MustCompile(`(?i)([0-9,]+)\s*(machines?|robots?|devices?)`),
			"area":     regexp.MustCompile(`(?i)(factory|plant|facility|warehouse)`),
		},
	}

	ip.logger.Info("Initialized intent patterns", "count", len(ip.patterns))
}

// normalizeText preprocesses input text for better pattern matching
func (ip *IntentParser) normalizeText(text string) string {
	// Convert to lowercase
	normalized := strings.ToLower(text)

	// Remove extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Expand common abbreviations
	replacements := map[string]string{
		"&":    "and",
		"w/":   "with",
		"w/o":  "without",
		"govt": "government",
		"mgmt": "management",
		"req":  "requirement",
		"reqs": "requirements",
	}

	for old, new := range replacements {
		normalized = strings.ReplaceAll(normalized, old, new)
	}

	return strings.TrimSpace(normalized)
}

// findBestMatch finds the best matching pattern for the input text
func (ip *IntentParser) findBestMatch(text string) (*IntentPattern, float64) {
	var bestPattern *IntentPattern
	var bestScore float64

	for _, pattern := range ip.patterns {
		if pattern.Regex.MatchString(text) {
			// Calculate confidence based on pattern strength and text context
			score := ip.calculateMatchScore(text, pattern)
			if score > bestScore {
				bestScore = score
				bestPattern = pattern
			}
		}
	}

	return bestPattern, bestScore
}

// calculateMatchScore calculates confidence score for pattern match
func (ip *IntentParser) calculateMatchScore(text string, pattern *IntentPattern) float64 {
	baseScore := pattern.Confidence

	// Boost score based on number of matches
	matches := pattern.Regex.FindAllString(text, -1)
	matchBoost := float64(len(matches)) * 0.05

	// Boost score if extractors find relevant data
	extractorBoost := 0.0
	for _, extractor := range pattern.Extractors {
		if extractor.MatchString(text) {
			extractorBoost += 0.1
		}
	}

	// Apply text length normalization
	lengthFactor := 1.0
	if len(text) < 50 {
		lengthFactor = 0.9 // Penalize very short texts
	} else if len(text) > 500 {
		lengthFactor = 0.95 // Slightly penalize very long texts
	}

	finalScore := (baseScore + matchBoost + extractorBoost) * lengthFactor
	if finalScore > 1.0 {
		finalScore = 1.0
	}

	return finalScore
}

// extractQoSProfile extracts QoS requirements from text using pattern template
func (ip *IntentParser) extractQoSProfile(text string, pattern *IntentPattern) (*QoSProfile, error) {
	profile := &QoSProfile{}

	// Start with template values
	if pattern.QoSTemplate != nil {
		*profile = *pattern.QoSTemplate
	}

	// Override with extracted values from text
	for field, extractor := range pattern.Extractors {
		matches := extractor.FindStringSubmatch(text)
		if len(matches) >= 2 {
			value, err := strconv.ParseFloat(matches[1], 64)
			if err != nil {
				continue
			}

			switch field {
			case "latency":
				profile.Latency.Target = &value
			case "bandwidth":
				// Convert units if needed
				if len(matches) >= 3 && strings.Contains(strings.ToLower(matches[2]), "gbps") {
					value *= 1000 // Convert Gbps to Mbps
				}
				profile.Bandwidth.Target = &value
			case "jitter":
				profile.Jitter.Target = &value
			}
		}
	}

	// Apply service-type specific adjustments
	ip.applyServiceTypeDefaults(pattern.ServiceType, profile)

	return profile, nil
}

// applyServiceTypeDefaults applies service-specific QoS defaults
func (ip *IntentParser) applyServiceTypeDefaults(serviceType ServiceType, profile *QoSProfile) {
	switch serviceType {
	case ServiceTypeURLLC:
		// Ultra-reliable, low-latency defaults
		if profile.Latency.Max == nil {
			profile.Latency.Max = floatPtr(10.0)
		}
		if profile.Reliability.Min == nil {
			profile.Reliability.Min = floatPtr(99.9)
		}
		if profile.PacketLoss.Max == nil {
			profile.PacketLoss.Max = floatPtr(0.1)
		}

	case ServiceTypeEMBB:
		// Enhanced mobile broadband defaults
		if profile.Bandwidth.Min == nil {
			profile.Bandwidth.Min = floatPtr(1.0)
		}
		if profile.Latency.Max == nil {
			profile.Latency.Max = floatPtr(50.0)
		}

	case ServiceTypeMmTC:
		// Massive machine type communication defaults
		if profile.Bandwidth.Max == nil {
			profile.Bandwidth.Max = floatPtr(1.0)
		}
		if profile.Latency.Max == nil {
			profile.Latency.Max = floatPtr(1000.0)
		}
	}
}

// extractConstraints extracts operational constraints from text
func (ip *IntentParser) extractConstraints(text string) ([]Constraint, error) {
	var constraints []Constraint

	// Location constraints
	locationRegex := regexp.MustCompile(`(?i)(in|at|near|within)\s+([a-zA-Z\s,]+?)(?:\s|$|,|\.)`)
	if matches := locationRegex.FindStringSubmatch(text); len(matches) >= 3 {
		constraints = append(constraints, Constraint{
			Type:        ConstraintLocation,
			Value:       strings.TrimSpace(matches[2]),
			Operator:    "eq",
			Description: "Geographic location requirement",
		})
	}

	// Latency constraints
	latencyRegex := regexp.MustCompile(`(?i)latency\s+(less than|under|below|<)\s+([0-9.]+)\s*(ms|milliseconds?)`)
	if matches := latencyRegex.FindStringSubmatch(text); len(matches) >= 3 {
		constraints = append(constraints, Constraint{
			Type:        ConstraintLatency,
			Value:       matches[2],
			Operator:    "lt",
			Description: "Maximum latency requirement",
			Mandatory:   true,
		})
	}

	// Cost constraints
	costRegex := regexp.MustCompile(`(?i)(budget|cost|price)\s+(under|below|less than|<)\s+([0-9.,]+)`)
	if matches := costRegex.FindStringSubmatch(text); len(matches) >= 4 {
		constraints = append(constraints, Constraint{
			Type:        ConstraintCost,
			Value:       matches[3],
			Operator:    "lt",
			Description: "Budget constraint",
		})
	}

	// Security/compliance constraints
	securityRegex := regexp.MustCompile(`(?i)(secure|encrypted|gdpr|hipaa|compliant|compliance|private)`)
	if securityRegex.MatchString(text) {
		constraints = append(constraints, Constraint{
			Type:        ConstraintSecurity,
			Value:       "required",
			Operator:    "eq",
			Description: "Security/compliance requirement",
			Mandatory:   true,
		})
	}

	return constraints, nil
}

// extractPriority extracts priority level from text
func (ip *IntentParser) extractPriority(text string) Priority {
	priorityRegex := regexp.MustCompile(`(?i)(critical|emergency|urgent|high priority|important)`)
	if priorityRegex.MatchString(text) {
		return PriorityCritical
	}

	highRegex := regexp.MustCompile(`(?i)(high|priority|important|asap)`)
	if highRegex.MatchString(text) {
		return PriorityHigh
	}

	lowRegex := regexp.MustCompile(`(?i)(low priority|when possible|eventually|background)`)
	if lowRegex.MatchString(text) {
		return PriorityLow
	}

	return PriorityMedium // Default
}

// generateSliceConfig creates slice configuration from parsed intent
func (ip *IntentParser) generateSliceConfig(intent *Intent) (*SliceConfig, error) {
	config := &SliceConfig{
		Name: fmt.Sprintf("%s-slice-%s", strings.ToLower(string(intent.ServiceType)), intent.ID[:8]),
		Type: intent.ServiceType,
		Coverage: Coverage{
			Indoor:  true,
			Outdoor: true,
			Density: "urban",
		},
		MobilityLevel: MobilityPedestrian, // Default
		DeviceTypes:   []string{"smartphone", "tablet"},
		Applications:  []ApplicationSpec{},
	}

	// Adjust configuration based on service type
	switch intent.ServiceType {
	case ServiceTypeURLLC:
		config.MobilityLevel = MobilityVehicular
		config.DeviceTypes = []string{"vehicle", "sensor", "actuator"}
		config.Applications = append(config.Applications, ApplicationSpec{
			Name:     "critical-control",
			Type:     "real-time",
			QoSClass: 1,
			TrafficPattern: TrafficPattern{
				Type:        "burst",
				Predictable: false,
			},
		})

	case ServiceTypeEMBB:
		config.DeviceTypes = []string{"smartphone", "tablet", "laptop", "ar-vr-device"}
		config.Applications = append(config.Applications, ApplicationSpec{
			Name:     "media-streaming",
			Type:     "bandwidth-intensive",
			QoSClass: 2,
			TrafficPattern: TrafficPattern{
				Type:        "constant",
				PeakHours:   []string{"19:00-23:00"},
				Predictable: true,
			},
		})

	case ServiceTypeMmTC:
		config.MobilityLevel = MobilityStationary
		config.DeviceTypes = []string{"sensor", "meter", "tracker"}
		config.Applications = append(config.Applications, ApplicationSpec{
			Name:     "telemetry",
			Type:     "periodic",
			QoSClass: 9,
			TrafficPattern: TrafficPattern{
				Type:        "periodic",
				Predictable: true,
			},
		})
	}

	// Extract coverage from constraints
	for _, constraint := range intent.Constraints {
		if constraint.Type == ConstraintLocation {
			config.Coverage.Areas = append(config.Coverage.Areas, constraint.Value)
		}
	}

	return config, nil
}

// validateQoSProfile validates QoS profile against service type constraints
func (ip *IntentParser) validateQoSProfile(serviceType ServiceType, profile *QoSProfile) error {
	switch serviceType {
	case ServiceTypeURLLC:
		// URLLC must have strict latency requirements
		if profile.Latency.Max != nil && *profile.Latency.Max > 50.0 {
			return fmt.Errorf("URLLC latency requirement too high: %f ms", *profile.Latency.Max)
		}
		if profile.Reliability.Min != nil && *profile.Reliability.Min < 99.0 {
			return fmt.Errorf("URLLC reliability requirement too low: %f%%", *profile.Reliability.Min)
		}

	case ServiceTypeEMBB:
		// eMBB must have adequate bandwidth
		if profile.Bandwidth.Min != nil && *profile.Bandwidth.Min < 1.0 {
			return fmt.Errorf("eMBB bandwidth requirement too low: %f Mbps", *profile.Bandwidth.Min)
		}

	case ServiceTypeMmTC:
		// mMTC should have conservative bandwidth requirements
		if profile.Bandwidth.Min != nil && *profile.Bandwidth.Min > 10.0 {
			ip.logger.Warn("mMTC bandwidth requirement seems high", "bandwidth", *profile.Bandwidth.Min)
		}
	}

	return nil
}

// Helper functions
func generateIntentID() string {
	return fmt.Sprintf("intent-%d", time.Now().UnixNano())
}

func floatPtr(f float64) *float64 {
	return &f
}