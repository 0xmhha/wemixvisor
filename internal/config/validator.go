package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// ValidationRule represents a configuration validation rule
type ValidationRule interface {
	Name() string
	Validate(cfg *Config) error
}

// Validator validates configuration
type Validator struct {
	rules  []ValidationRule
	logger *logger.Logger
}

// NewValidator creates a new configuration validator
func NewValidator(logger *logger.Logger) *Validator {
	v := &Validator{
		logger: logger,
	}

	// Register default validation rules
	v.registerDefaultRules()

	return v
}

// Validate validates the configuration
func (v *Validator) Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Run all validation rules
	var errors []string
	for _, rule := range v.rules {
		if err := rule.Validate(cfg); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", rule.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// ValidateField validates a specific configuration field
func (v *Validator) ValidateField(field string, value interface{}) error {
	// Implement field-specific validation
	switch field {
	case "home":
		return v.validatePath(value)
	case "name":
		return v.validateBinaryName(value)
	case "rpc_port", "ws_port", "p2p_port":
		return v.validatePort(value)
	case "network_id":
		return v.validateNetworkID(value)
	case "chain_id":
		return v.validateChainID(value)
	case "max_peers":
		return v.validateMaxPeers(value)
	default:
		// Unknown field, no specific validation
		return nil
	}
}

// AddRule adds a custom validation rule
func (v *Validator) AddRule(rule ValidationRule) {
	v.rules = append(v.rules, rule)
}

// registerDefaultRules registers default validation rules
func (v *Validator) registerDefaultRules() {
	v.rules = []ValidationRule{
		&pathValidationRule{},
		&binaryValidationRule{},
		&portValidationRule{},
		&networkValidationRule{},
		&resourceValidationRule{},
		&securityValidationRule{},
		&compatibilityValidationRule{},
	}
}

// validatePath validates a path value
func (v *Validator) validatePath(value interface{}) error {
	path, ok := value.(string)
	if !ok {
		return fmt.Errorf("path must be a string")
	}

	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(os.Getenv("HOME"), path[2:])
	}

	// Check if parent directory exists
	parent := filepath.Dir(path)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		return fmt.Errorf("parent directory does not exist: %s", parent)
	}

	return nil
}

// validateBinaryName validates a binary name
func (v *Validator) validateBinaryName(value interface{}) error {
	name, ok := value.(string)
	if !ok {
		return fmt.Errorf("binary name must be a string")
	}

	if name == "" {
		return fmt.Errorf("binary name cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("binary name contains invalid characters")
	}

	return nil
}

// validatePort validates a port number
func (v *Validator) validatePort(value interface{}) error {
	port, ok := value.(int)
	if !ok {
		// Try to convert from other numeric types
		if p, ok := value.(int64); ok {
			port = int(p)
		} else if p, ok := value.(float64); ok {
			port = int(p)
		} else {
			return fmt.Errorf("port must be a number")
		}
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// validateNetworkID validates a network ID
func (v *Validator) validateNetworkID(value interface{}) error {
	networkID, ok := value.(uint64)
	if !ok {
		// Try to convert from other numeric types
		if id, ok := value.(int); ok {
			networkID = uint64(id)
		} else if id, ok := value.(float64); ok {
			networkID = uint64(id)
		} else {
			return fmt.Errorf("network ID must be a number")
		}
	}

	if networkID == 0 {
		return fmt.Errorf("network ID cannot be 0")
	}

	return nil
}

// validateChainID validates a chain ID
func (v *Validator) validateChainID(value interface{}) error {
	chainID, ok := value.(string)
	if !ok {
		return fmt.Errorf("chain ID must be a string")
	}

	if chainID == "" {
		return fmt.Errorf("chain ID cannot be empty")
	}

	// Validate format (should be numeric string for WEMIX)
	if !regexp.MustCompile(`^\d+$`).MatchString(chainID) {
		return fmt.Errorf("chain ID must be a numeric string")
	}

	return nil
}

// validateMaxPeers validates max peers setting
func (v *Validator) validateMaxPeers(value interface{}) error {
	maxPeers, ok := value.(int)
	if !ok {
		if p, ok := value.(float64); ok {
			maxPeers = int(p)
		} else {
			return fmt.Errorf("max peers must be a number")
		}
	}

	if maxPeers < 0 {
		return fmt.Errorf("max peers cannot be negative")
	}

	if maxPeers > 200 {
		return fmt.Errorf("max peers should not exceed 200")
	}

	return nil
}

// pathValidationRule validates path-related configuration
type pathValidationRule struct{}

func (r *pathValidationRule) Name() string {
	return "PathValidation"
}

func (r *pathValidationRule) Validate(cfg *Config) error {
	// Validate home directory
	if cfg.Home == "" {
		return fmt.Errorf("home directory is required")
	}

	// Expand and check home directory
	home := cfg.Home
	if strings.HasPrefix(home, "~/") {
		home = filepath.Join(os.Getenv("HOME"), home[2:])
	}

	// Check if directory exists or can be created
	if info, err := os.Stat(home); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("home path exists but is not a directory: %s", home)
		}
	}

	// Validate data backup path if specified
	if cfg.DataBackupPath != "" {
		backupPath := cfg.DataBackupPath
		if strings.HasPrefix(backupPath, "~/") {
			backupPath = filepath.Join(os.Getenv("HOME"), backupPath[2:])
		}

		// Check if parent directory exists
		parent := filepath.Dir(backupPath)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			return fmt.Errorf("backup path parent directory does not exist: %s", parent)
		}
	}

	return nil
}

// binaryValidationRule validates binary-related configuration
type binaryValidationRule struct{}

func (r *binaryValidationRule) Name() string {
	return "BinaryValidation"
}

func (r *binaryValidationRule) Validate(cfg *Config) error {
	// Validate binary name
	if cfg.Name == "" {
		return fmt.Errorf("daemon name is required")
	}

	// Check for invalid characters
	if strings.ContainsAny(cfg.Name, "/\\:*?\"<>|") {
		return fmt.Errorf("daemon name contains invalid characters: %s", cfg.Name)
	}

	return nil
}

// portValidationRule validates network port configuration
type portValidationRule struct{}

func (r *portValidationRule) Name() string {
	return "PortValidation"
}

func (r *portValidationRule) Validate(cfg *Config) error {
	// Validate RPC port if specified
	if cfg.RPCPort > 0 {
		if cfg.RPCPort > 65535 {
			return fmt.Errorf("RPC port %d is invalid", cfg.RPCPort)
		}

		// Check if port is available
		if !isPortAvailable(cfg.RPCPort) {
			return fmt.Errorf("RPC port %d is already in use", cfg.RPCPort)
		}
	}

	return nil
}

// networkValidationRule validates network configuration
type networkValidationRule struct{}

func (r *networkValidationRule) Name() string {
	return "NetworkValidation"
}

func (r *networkValidationRule) Validate(cfg *Config) error {
	// Validate RPC address if specified
	if cfg.RPCAddress != "" {
		// Parse the address
		host, port, err := net.SplitHostPort(cfg.RPCAddress)
		if err != nil {
			// Try without port
			if net.ParseIP(cfg.RPCAddress) == nil {
				return fmt.Errorf("invalid RPC address: %s", cfg.RPCAddress)
			}
		} else {
			// Validate host
			if host != "" && host != "localhost" && net.ParseIP(host) == nil {
				// Try to resolve as hostname
				if _, err := net.LookupHost(host); err != nil {
					return fmt.Errorf("cannot resolve RPC host: %s", host)
				}
			}

			// Validate port
			if port != "" {
				var portNum int
				if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
					return fmt.Errorf("invalid RPC port: %s", port)
				}
				if portNum < 1 || portNum > 65535 {
					return fmt.Errorf("RPC port %d out of range", portNum)
				}
			}
		}
	}

	return nil
}

// resourceValidationRule validates resource limits
type resourceValidationRule struct{}

func (r *resourceValidationRule) Name() string {
	return "ResourceValidation"
}

func (r *resourceValidationRule) Validate(cfg *Config) error {
	// Validate shutdown grace period
	if cfg.ShutdownGrace < 0 {
		return fmt.Errorf("shutdown grace period cannot be negative")
	}

	if cfg.ShutdownGrace > 10*time.Minute {
		return fmt.Errorf("shutdown grace period too long (max 10 minutes)")
	}

	// Validate poll interval
	if cfg.PollInterval < 0 {
		return fmt.Errorf("poll interval cannot be negative")
	}

	if cfg.PollInterval > 0 && cfg.PollInterval < 100*time.Millisecond {
		return fmt.Errorf("poll interval too short (min 100ms)")
	}

	// Validate restart delay
	if cfg.RestartDelay < 0 {
		return fmt.Errorf("restart delay cannot be negative")
	}

	// Validate health check interval
	if cfg.HealthCheckInterval < 0 {
		return fmt.Errorf("health check interval cannot be negative")
	}

	if cfg.HealthCheckInterval > 0 && cfg.HealthCheckInterval < 5*time.Second {
		return fmt.Errorf("health check interval too short (min 5s)")
	}

	// Validate metrics interval
	if cfg.MetricsInterval < 0 {
		return fmt.Errorf("metrics interval cannot be negative")
	}

	if cfg.MetricsInterval > 0 && cfg.MetricsInterval < 10*time.Second {
		return fmt.Errorf("metrics interval too short (min 10s)")
	}

	// Validate max restarts
	if cfg.MaxRestarts < 0 {
		return fmt.Errorf("max restarts cannot be negative")
	}

	if cfg.MaxRestarts > 100 {
		return fmt.Errorf("max restarts too high (max 100)")
	}

	// Validate pre-upgrade max retries
	if cfg.PreUpgradeMaxRetries < 0 {
		return fmt.Errorf("pre-upgrade max retries cannot be negative")
	}

	if cfg.PreUpgradeMaxRetries > 10 {
		return fmt.Errorf("pre-upgrade max retries too high (max 10)")
	}

	return nil
}

// securityValidationRule validates security settings
type securityValidationRule struct{}

func (r *securityValidationRule) Name() string {
	return "SecurityValidation"
}

func (r *securityValidationRule) Validate(cfg *Config) error {
	// Warn about unsafe settings
	if cfg.UnsafeSkipBackup {
		// This is a warning, not an error
		// Logger would log a warning here
	}

	if cfg.UnsafeSkipChecksum && cfg.AllowDownloadBinaries {
		return fmt.Errorf("downloading binaries without checksum verification is extremely unsafe")
	}

	// Validate custom pre-upgrade script if specified
	if cfg.CustomPreUpgrade != "" {
		// Check if file exists
		if _, err := os.Stat(cfg.CustomPreUpgrade); os.IsNotExist(err) {
			return fmt.Errorf("custom pre-upgrade script not found: %s", cfg.CustomPreUpgrade)
		}

		// Check if executable
		info, err := os.Stat(cfg.CustomPreUpgrade)
		if err != nil {
			return fmt.Errorf("cannot stat pre-upgrade script: %w", err)
		}

		if info.Mode()&0111 == 0 {
			return fmt.Errorf("pre-upgrade script is not executable: %s", cfg.CustomPreUpgrade)
		}
	}

	return nil
}

// compatibilityValidationRule validates version compatibility
type compatibilityValidationRule struct{}

func (r *compatibilityValidationRule) Name() string {
	return "CompatibilityValidation"
}

func (r *compatibilityValidationRule) Validate(cfg *Config) error {
	// Check for known incompatible settings combinations
	if cfg.ValidatorMode && cfg.DisableRecase {
		return fmt.Errorf("validator mode requires recase to be enabled")
	}

	// Check time format
	if cfg.TimeFormatLogs != "" {
		validFormats := []string{"kitchen", "rfc3339", "rfc3339nano", "iso8601"}
		valid := false
		for _, format := range validFormats {
			if cfg.TimeFormatLogs == format {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid time format: %s (valid: %s)", cfg.TimeFormatLogs, strings.Join(validFormats, ", "))
		}
	}

	return nil
}

// isPortAvailable checks if a port is available
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}