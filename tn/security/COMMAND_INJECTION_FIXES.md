# Command Injection Vulnerability Fixes

## Overview

This document describes the comprehensive security fixes implemented to address command injection vulnerabilities in the Transport Network (TN) Agent components.

## Fixed Vulnerabilities

### 1. tn/agent/main.go Lines 228 & 259

**Original Issue**: Subprocess launched with variables without proper validation
**Files**: `tn/agent/main.go` functions `executeCommand()` and `executeCommandOutput()`

**Security Fixes Implemented**:

1. **Enhanced Input Validation**:
   - Added validation of entire command string before parsing
   - Individual argument validation for each command parameter
   - Command allowlisting with strict whitelist of permitted commands

2. **Multi-Layer Security**:
   - Command string validation using `security.ValidateCommandArgument()`
   - Command allowlisting (only `ip`, `tc`, `bridge`, `ping`, `iperf3` allowed)
   - Command-specific validators for `tc`, `ip`, and `iperf3` commands
   - Secure subprocess execution with `security.SecureExecute()` and `security.SecureExecuteWithValidation()`

3. **Enhanced Error Handling**:
   - Sanitized error messages to prevent information leakage
   - Proper timeout handling for all subprocess executions

### 2. tn/agent/pkg/iperf.go Lines 198 & 388

**Original Issue**: Subprocess launched with variables without proper validation
**Files**: `tn/agent/pkg/iperf.go` functions `StartServer()` and `RunTest()`

**Security Fixes Implemented**:

1. **Comprehensive Input Validation**:
   - Port validation with `security.ValidatePort()`
   - IP address validation with `security.ValidateIPAddress()`
   - Bandwidth validation with `security.ValidateBandwidth()`
   - Individual argument validation for all iperf3 parameters

2. **Secure Execution Framework**:
   - All iperf3 commands use `security.SecureExecute()` or `security.SecureExecuteWithValidation()`
   - Custom iperf3 argument validation with `security.ValidateIPerfArgs()`
   - Timeout protection for all network operations

3. **Process Management**:
   - Secure process identification using `pgrep` with validated patterns
   - Process termination using `pkill` with validated arguments
   - Resource limits and cleanup procedures

## Security Framework Components

### Input Validation Layer
- **Command Argument Validation**: Prevents shell metacharacters
- **Network Parameter Validation**: IP addresses, ports, bandwidth values
- **File Path Validation**: Prevents path traversal attacks
- **Interface Name Validation**: Network interface name format checking

### Command Execution Layer
- **Allowlist-based Command Control**: Only predefined safe commands allowed
- **Argument Sanitization**: All arguments validated before execution
- **Command-specific Validators**: Specialized validation for tc, ip, iperf3
- **Secure Subprocess Execution**: Protected execution environment

### Process Isolation Layer
- **Timeout Controls**: All subprocess operations have timeouts
- **Environment Sanitization**: Clean execution environment
- **Resource Limits**: Output size and execution time limits
- **Process Attribute Security**: Platform-specific security attributes

## Security Validation

### Command Allowlist
```go
allowedCommands := []string{"ip", "tc", "bridge", "ping", "iperf3"}
```

### Input Validation Examples
```go
// Command string validation
if err := security.ValidateCommandArgument(cmdStr); err != nil {
    return fmt.Errorf("unsafe command string: %w", err)
}

// Individual argument validation
for i, arg := range parts[1:] {
    if err := security.ValidateCommandArgument(arg); err != nil {
        return fmt.Errorf("invalid argument %d: %w", i+1, err)
    }
}
```

### Secure Execution Pattern
```go
// Command-specific validation
switch parts[0] {
case "tc":
    customValidator = security.ValidateTCArgs
case "ip":
    customValidator = security.ValidateIPArgs
case "iperf3":
    customValidator = security.ValidateIPerfArgs
default:
    customValidator = nil
}

// Secure execution with validation
if customValidator != nil {
    _, err = security.SecureExecuteWithValidation(ctx, parts[0], customValidator, parts[1:]...)
} else {
    _, err = security.SecureExecute(ctx, parts[0], parts[1:]...)
}
```

## Testing and Verification

All command injection vulnerabilities have been resolved through:

1. **Static Analysis**: No direct `exec.Command` or `exec.CommandContext` usage found
2. **Input Validation**: All user-controlled input is validated before subprocess execution
3. **Secure Execution**: All subprocess operations use the security framework
4. **Allowlist Control**: Only whitelisted commands are permitted for execution

## Benefits

1. **Command Injection Prevention**: Multiple validation layers prevent malicious command injection
2. **Defense in Depth**: Layered security approach with multiple checkpoints
3. **Audit Trail**: Comprehensive logging of all command executions
4. **Resource Protection**: Timeout and size limits prevent resource exhaustion
5. **Error Handling**: Secure error messages prevent information disclosure

## Compliance

These fixes ensure compliance with security best practices:
- OWASP Command Injection Prevention
- Secure coding standards for Go applications
- Defense in depth security architecture
- Input validation and sanitization requirements