# NLP Intent to QoS Translation Guide

## Overview
This guide defines how natural language intents are mapped to quantitative QoS parameters for network slice configuration. The system translates user intents into JSON objects validated against `schema.json`.

## QoS Parameter Bounds
- **Bandwidth**: 1-5 Mbps (integer or decimal)
- **Latency**: 1-10 ms (integer or decimal)
- **Jitter**: 0-5 ms (optional)
- **Packet Loss**: 0-1% (optional)
- **Reliability**: 95-99.999% (optional)

## Canonical Intent Mappings

### 1. Enhanced Mobile Broadband (eMBB)
**Intent Keywords**: "high bandwidth", "video streaming", "content delivery", "multimedia"
**Mapping**:
```json
{
  "bandwidth": 5,
  "latency": 9,
  "slice_type": "eMBB"
}
```
**Use Cases**: 4K video streaming, AR/VR applications, high-speed downloads

### 2. Ultra-Reliable Low Latency Communication (uRLLC)
**Intent Keywords**: "low latency", "real-time", "mission critical", "industrial", "emergency"
**Mapping**:
```json
{
  "bandwidth": 1,
  "latency": 1,
  "slice_type": "uRLLC"
}
```
**Use Cases**: Industrial automation, remote surgery, autonomous vehicles, emergency services

### 3. Balanced Performance
**Intent Keywords**: "balanced", "general purpose", "standard", "typical"
**Mapping**:
```json
{
  "bandwidth": 3,
  "latency": 9,
  "slice_type": "balanced"
}
```
**Use Cases**: Enterprise applications, web browsing, standard IoT

## Intent Processing Rules

### Rule 1: Pattern Matching
The system uses keyword-based pattern matching to classify intents:

**eMBB Keywords**: high bandwidth, embb, video streaming, multimedia, content delivery, 4k video, streaming, download, throughput, ar, vr, augmented reality, virtual reality

**uRLLC Keywords**: low latency, urllc, real-time, mission critical, industrial, emergency, control, robotics, automation, ultra-low, critical, responsive, immediate

**Balanced Keywords**: balanced, general, standard, typical, normal, enterprise, business, general purpose, medium, moderate, average, default

### Rule 2: Scoring and Selection
- Each keyword match increases the pattern score
- The pattern with the highest score is selected
- If no patterns match, defaults to balanced profile
- Multiple keyword matches strengthen pattern confidence

### Rule 3: Enhancement Rules
Additional QoS parameters are added based on context:

**Reliability Enhancement**: Added when intent contains 'critical', 'emergency', 'mission', or 'industrial'
- Sets reliability to 99.99%

**Jitter Constraints**: Added for 'real-time', 'gaming', 'voice', or 'video call'
- Sets jitter to 1.0 ms

**Packet Loss Constraints**: Added for 'streaming', 'video', 'voice', or 'multimedia'
- Sets packet_loss to 0.1%

### Rule 4: Validation and Error Handling
- All QoS parameters validated against schema.json
- Out-of-range values trigger validation errors
- Missing required fields (bandwidth, latency) cause failures
- Invalid slice_type values are rejected

### Rule 5: CLI Processing
The `run_intents.py` module processes files with these rules:
- One intent per line
- Lines starting with # are treated as comments
- Empty lines are skipped
- Errors are logged but don't stop processing
- Output format is JSONL (one JSON object per line)

## Example Intent Translations

### Example 1: Video Streaming Service
**Intent**: "Create a slice for HD video streaming service"
**Analysis**:
- Primary: High bandwidth for video
- Secondary: Moderate latency acceptable
**Output**:
```json
{
  "bandwidth": 5,
  "latency": 9,
  "slice_type": "eMBB",
  "packet_loss": 0.1
}
```

### Example 2: Industrial IoT Control
**Intent**: "Need ultra-low latency slice for factory automation"
**Analysis**:
- Primary: Minimal latency critical
- Secondary: Low bandwidth sufficient
**Output**:
```json
{
  "bandwidth": 1,
  "latency": 1,
  "slice_type": "uRLLC",
  "reliability": 99.99
}
```

### Example 3: Enterprise Application
**Intent**: "Standard enterprise application hosting"
**Analysis**:
- No specific emphasis
- Balanced requirements
**Output**:
```json
{
  "bandwidth": 3,
  "latency": 9,
  "slice_type": "balanced"
}
```

## CLI Usage

The `run_intents.py` module provides a command-line interface for processing intent files:

```bash
# Basic usage
python run_intents.py fixtures/intents.txt

# Output to file
python run_intents.py fixtures/intents.txt --output results.jsonl

# Verbose logging
python run_intents.py fixtures/intents.txt --verbose

# Validation only (no output)
python run_intents.py fixtures/intents.txt --validate-only

# Custom schema
python run_intents.py fixtures/intents.txt --schema custom_schema.json
```

## Validation Process

1. **Parse Intent**: Extract keywords and requirements using pattern matching
2. **Score Patterns**: Count keyword matches for each intent type
3. **Select Best Match**: Choose highest-scoring pattern or default to balanced
4. **Enhance Parameters**: Add context-specific QoS parameters
5. **Validate Schema**: Check against `schema.json` constraints
6. **Return Result**: Valid JSON or validation errors in JSONL format

## Error Handling

### Out of Range Errors
- Bandwidth < 1 or > 5: "Bandwidth must be between 1 and 5 Mbps"
- Latency < 1 or > 10: "Latency must be between 1 and 10 ms"

### Invalid Types
- Non-numeric values: "Parameter must be a number"
- Invalid slice_type: "Slice type must be one of: eMBB, uRLLC, mIoT, balanced"

### Missing Required Fields
- "Missing required field: bandwidth"
- "Missing required field: latency"