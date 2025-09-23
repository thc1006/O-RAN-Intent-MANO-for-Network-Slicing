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

### Rule 1: Bandwidth Priority
When intent emphasizes throughput/bandwidth:
- "maximum bandwidth" → bandwidth: 5
- "high throughput" → bandwidth: 4-5
- "moderate bandwidth" → bandwidth: 2-3
- "minimal bandwidth" → bandwidth: 1

### Rule 2: Latency Priority
When intent emphasizes responsiveness:
- "ultra-low latency" → latency: 1
- "low latency" → latency: 1-3
- "moderate latency" → latency: 4-7
- "latency tolerant" → latency: 8-10

### Rule 3: Trade-off Resolution
When conflicting requirements exist:
1. Identify primary requirement (first mentioned or emphasized)
2. Apply primary requirement at maximum
3. Adjust secondary within remaining budget

### Rule 4: Edge Cases
- Exact boundary values (1, 5, 10) are valid
- Decimal values are rounded to 1 decimal place
- Missing explicit values default to balanced profile

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
  "slice_type": "eMBB"
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

## Validation Process

1. **Parse Intent**: Extract keywords and requirements
2. **Map to Parameters**: Apply rules to generate QoS values
3. **Validate Schema**: Check against `schema.json` constraints
4. **Return Result**: Valid JSON or validation errors

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