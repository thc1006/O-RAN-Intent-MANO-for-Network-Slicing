package tests

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseFloat64WithUnits parses a string containing a numeric value and optional unit suffix.
// It handles various formats including plain numbers, scientific notation, and values with units.
//
// Examples:
//   - "123" → (123.0, "", nil)
//   - "4.5 Mbps" → (4.5, "Mbps", nil)
//   - "1.5e3 Mbps" → (1500.0, "Mbps", nil)
//   - "10ms" → (10.0, "ms", nil)
//
// Returns:
//   - value: The parsed numeric value as float64
//   - unit: The unit suffix (empty string if no unit)
//   - error: Error if parsing fails
func parseFloat64WithUnits(input string) (float64, string, error) {
	// 1. Trim whitespace
	input = strings.TrimSpace(input)

	// 2. Check empty input
	if input == "" {
		return 0, "", fmt.Errorf("empty input")
	}

	// 3. Use regex to extract number and unit
	// Pattern explanation:
	// ^                          - Start of string
	// ([+-]?                     - Optional sign
	//   (?:\d+\.?\d*|\.\d+)      - Number: digits with optional decimal OR decimal with digits
	//   (?:[eE][+-]?\d+)?        - Optional scientific notation exponent
	// )
	// \s*                        - Optional whitespace between number and unit
	// ([a-zA-Z%]*)               - Optional unit (letters or percentage)
	// $                          - End of string
	re := regexp.MustCompile(`^([+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?)\s*([a-zA-Z%]*)$`)
	matches := re.FindStringSubmatch(input)

	// 4. Check if input contains only letters (unit only, no number)
	if matches == nil {
		// Check if it's only letters/unit (missing value case)
		onlyLetters := regexp.MustCompile(`^[a-zA-Z%]+$`)
		if onlyLetters.MatchString(input) {
			// Check if it's a known unit (missing value) vs unknown text (invalid format)
			normalized := normalizeUnit(input)
			lowerInput := strings.ToLower(input)

			// List of known units
			knownUnits := map[string]bool{
				"bps": true, "kbps": true, "mbps": true, "gbps": true, "tbps": true,
				"ns": true, "us": true, "ms": true, "s": true, "m": true, "h": true,
				"b": true, "kb": true, "mb": true, "gb": true, "tb": true, "pb": true,
				"db": true, "%": true,
			}

			if knownUnits[lowerInput] || normalized != input {
				// It's a known unit, so the value is missing
				return 0, "", fmt.Errorf("missing value in: %s", input)
			}
			// It's just random text
			return 0, "", fmt.Errorf("invalid format: %s", input)
		}

		// Check if it contains invalid characters in the number part
		if len(strings.Fields(input)) > 0 {
			hasInvalidChars := regexp.MustCompile(`[^0-9+\-\.eE\s]`).MatchString(strings.Fields(input)[0])
			if hasInvalidChars {
				return 0, "", fmt.Errorf("invalid number in: %s", input)
			}
		}

		return 0, "", fmt.Errorf("invalid format: %s", input)
	}

	// 5. Parse the numeric part
	numStr := matches[1]
	if numStr == "" {
		return 0, "", fmt.Errorf("missing value in: %s", input)
	}

	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid number: %s", numStr)
	}

	// 6. Extract and normalize the unit
	unit := matches[2]
	if unit != "" {
		unit = normalizeUnit(unit)
	}

	return value, unit, nil
}

// normalizeUnit normalizes unit strings to standard casing.
// This ensures consistent unit representation regardless of input casing.
//
// Examples:
//   - "mbps" → "Mbps"
//   - "MBPS" → "Mbps"
//   - "MS" → "ms"
//   - "gb" → "GB"
func normalizeUnit(unit string) string {
	if unit == "" {
		return ""
	}

	// Define standard unit formats
	unitMap := map[string]string{
		// Bandwidth units (bits per second)
		"bps":  "bps",
		"kbps": "Kbps",
		"mbps": "Mbps",
		"gbps": "Gbps",
		"tbps": "Tbps",

		// Time units
		"ns": "ns",
		"us": "us",
		"ms": "ms",
		"s":  "s",
		"m":  "m",
		"h":  "h",

		// Memory/Storage units (bytes)
		"b":  "B",
		"kb": "KB",
		"mb": "MB",
		"gb": "GB",
		"tb": "TB",
		"pb": "PB",

		// Other common units
		"db": "dB",
		"%":  "%",
	}

	// Convert to lowercase for lookup
	lowerUnit := strings.ToLower(unit)

	// Check if we have a standardized version
	if normalized, exists := unitMap[lowerUnit]; exists {
		return normalized
	}

	// If no mapping exists, return as-is (for unknown units)
	return unit
}

// convertToBaseUnit converts a value from one unit to another within the same category.
// It handles conversions for bandwidth, time, and memory units.
//
// Unit Categories:
//   - Bandwidth: bps, Kbps, Mbps, Gbps, Tbps
//   - Time: ns, us, ms, s, m, h
//   - Memory: B, KB, MB, GB, TB, PB
//
// Returns:
//   - result: The converted value
//   - error: Error if units are incompatible or unknown
func convertToBaseUnit(value float64, fromUnit, toUnit string) (float64, error) {
	// Normalize units first
	fromUnit = normalizeUnit(fromUnit)
	toUnit = normalizeUnit(toUnit)

	// If units are the same, no conversion needed
	if fromUnit == toUnit {
		return value, nil
	}

	// Define unit categories and conversion factors (to base unit)
	type unitCategory struct {
		baseUnit string
		factors  map[string]float64
	}

	categories := []unitCategory{
		// Bandwidth (base: bps)
		{
			baseUnit: "bps",
			factors: map[string]float64{
				"bps":  1.0,
				"Kbps": 1e3,
				"Mbps": 1e6,
				"Gbps": 1e9,
				"Tbps": 1e12,
			},
		},
		// Time (base: s)
		{
			baseUnit: "s",
			factors: map[string]float64{
				"ns": 1e-9,
				"us": 1e-6,
				"ms": 1e-3,
				"s":  1.0,
				"m":  60.0,
				"h":  3600.0,
			},
		},
		// Memory (base: B) - using binary (1024) conversion
		{
			baseUnit: "B",
			factors: map[string]float64{
				"B":  1.0,
				"KB": 1024.0,
				"MB": 1024.0 * 1024.0,
				"GB": 1024.0 * 1024.0 * 1024.0,
				"TB": 1024.0 * 1024.0 * 1024.0 * 1024.0,
				"PB": 1024.0 * 1024.0 * 1024.0 * 1024.0 * 1024.0,
			},
		},
	}

	// Find which category both units belong to
	var fromFactor, toFactor float64
	var foundCategory bool

	for _, category := range categories {
		fromF, fromExists := category.factors[fromUnit]
		toF, toExists := category.factors[toUnit]

		if fromExists && toExists {
			fromFactor = fromF
			toFactor = toF
			foundCategory = true
			break
		}
	}

	if !foundCategory {
		// Check if units exist in different categories (incompatible)
		fromExists := false
		toExists := false

		for _, category := range categories {
			if _, exists := category.factors[fromUnit]; exists {
				fromExists = true
			}
			if _, exists := category.factors[toUnit]; exists {
				toExists = true
			}
		}

		if fromExists && toExists {
			return 0, fmt.Errorf("incompatible units: cannot convert %s to %s", fromUnit, toUnit)
		} else if !fromExists {
			return 0, fmt.Errorf("unknown source unit: %s", fromUnit)
		} else if !toExists {
			return 0, fmt.Errorf("unknown target unit: %s", toUnit)
		}

		return 0, fmt.Errorf("unsupported unit conversion: %s to %s", fromUnit, toUnit)
	}

	// Convert: value_in_base = value * fromFactor
	// result = value_in_base / toFactor
	result := value * fromFactor / toFactor

	return result, nil
}