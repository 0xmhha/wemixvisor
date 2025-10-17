package config

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/wemix/wemixvisor/pkg/logger"
)

//go:embed templates/*.toml templates/*.yaml
var templatesFS embed.FS

// Template represents a configuration template
type Template struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Network     string                 `json:"network"`
	Values      map[string]interface{} `json:"values"`
}

// TemplateManager manages configuration templates
type TemplateManager struct {
	templates map[string]*Template
	logger    *logger.Logger
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(logger *logger.Logger) *TemplateManager {
	tm := &TemplateManager{
		templates: make(map[string]*Template),
		logger:    logger,
	}

	// Load embedded templates
	if err := tm.loadEmbeddedTemplates(); err != nil {
		logger.Warn("failed to load embedded templates: " + err.Error())
	}

	// Load default templates
	tm.loadDefaultTemplates()

	return tm
}

// GetTemplate returns a template by name
func (tm *TemplateManager) GetTemplate(name string) (*Template, error) {
	template, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}
	return template, nil
}

// ListTemplates returns all available templates
func (tm *TemplateManager) ListTemplates() []*Template {
	templates := make([]*Template, 0, len(tm.templates))
	for _, t := range tm.templates {
		templates = append(templates, t)
	}
	return templates
}

// AddTemplate adds a new template
func (tm *TemplateManager) AddTemplate(template *Template) error {
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	tm.templates[template.Name] = template
	tm.logger.Info("added template: " + template.Name)
	return nil
}

// RemoveTemplate removes a template
func (tm *TemplateManager) RemoveTemplate(name string) error {
	if _, exists := tm.templates[name]; !exists {
		return fmt.Errorf("template %s not found", name)
	}

	delete(tm.templates, name)
	tm.logger.Info("removed template: " + name)
	return nil
}

// LoadFromFile loads a template from a file
func (tm *TemplateManager) LoadFromFile(path string) error {
	// Determine format from extension
	ext := filepath.Ext(path)

	var template Template
	var err error

	switch ext {
	case ".toml":
		err = tm.loadTOMLTemplate(path, &template)
	case ".yaml", ".yml":
		err = tm.loadYAMLTemplate(path, &template)
	default:
		return fmt.Errorf("unsupported template format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to load template from %s: %w", path, err)
	}

	// Set name from filename if not specified
	if template.Name == "" {
		template.Name = strings.TrimSuffix(filepath.Base(path), ext)
	}

	return tm.AddTemplate(&template)
}

// loadEmbeddedTemplates loads templates from embedded filesystem
func (tm *TemplateManager) loadEmbeddedTemplates() error {
	return fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read template file
		data, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		// Parse based on extension
		ext := filepath.Ext(path)
		var template Template

		switch ext {
		case ".toml":
			if err := toml.Unmarshal(data, &template); err != nil {
				return fmt.Errorf("failed to parse TOML template %s: %w", path, err)
			}
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &template); err != nil {
				return fmt.Errorf("failed to parse YAML template %s: %w", path, err)
			}
		default:
			return nil // Skip unsupported formats
		}

		// Set name from filename if not specified
		if template.Name == "" {
			template.Name = strings.TrimSuffix(filepath.Base(path), ext)
		}

		tm.templates[template.Name] = &template
		return nil
	})
}

// loadDefaultTemplates loads hardcoded default templates
func (tm *TemplateManager) loadDefaultTemplates() {
	// Mainnet template
	tm.templates["mainnet"] = &Template{
		Name:        "mainnet",
		Description: "WEMIX3.0 Mainnet configuration",
		Network:     "mainnet",
		Values: map[string]interface{}{
			"network_id": uint64(1111),
			"chain_id":   "1111",
			"rpc_port":   8588,
			"ws_port":    8598,
			"p2p_port":   8589,
			"bootnodes": []string{
				"enode://abc123...@mainnet-boot1.wemix.com:8589",
				"enode://def456...@mainnet-boot2.wemix.com:8589",
			},
			"max_peers":      50,
			"validator_mode": false,
			"data_dir":       "~/.wemix/mainnet",
		},
	}

	// Testnet template
	tm.templates["testnet"] = &Template{
		Name:        "testnet",
		Description: "WEMIX3.0 Testnet configuration",
		Network:     "testnet",
		Values: map[string]interface{}{
			"network_id": uint64(1112),
			"chain_id":   "1112",
			"rpc_port":   8545,
			"ws_port":    8546,
			"p2p_port":   30303,
			"bootnodes": []string{
				"enode://test123...@testnet-boot1.wemix.com:30303",
				"enode://test456...@testnet-boot2.wemix.com:30303",
			},
			"max_peers":      25,
			"validator_mode": false,
			"data_dir":       "~/.wemix/testnet",
		},
	}

	// Devnet template
	tm.templates["devnet"] = &Template{
		Name:        "devnet",
		Description: "Local development network configuration",
		Network:     "devnet",
		Values: map[string]interface{}{
			"network_id":     uint64(1337),
			"chain_id":       "1337",
			"rpc_port":       8545,
			"ws_port":        8546,
			"p2p_port":       30303,
			"bootnodes":      []string{},
			"max_peers":      10,
			"validator_mode": false,
			"data_dir":       "~/.wemix/devnet",
			"extra_flags": []string{
				"--dev",
				"--dev.period=1",
			},
		},
	}

	// Validator template
	tm.templates["validator"] = &Template{
		Name:        "validator",
		Description: "Validator node configuration",
		Network:     "custom",
		Values: map[string]interface{}{
			"validator_mode":      true,
			"max_peers":          100,
			"health_check_interval": "10s",
			"metrics_enabled":     true,
			"metrics_interval":    "30s",
			"auto_backup":        true,
			"max_restarts":       3,
			"shutdown_grace":     "60s",
		},
	}

	// Archive template
	tm.templates["archive"] = &Template{
		Name:        "archive",
		Description: "Archive node configuration",
		Network:     "custom",
		Values: map[string]interface{}{
			"extra_flags": []string{
				"--syncmode=full",
				"--gcmode=archive",
				"--cache=4096",
			},
			"max_peers":       50,
			"metrics_enabled": true,
			"auto_backup":    false, // Archive nodes are typically too large to backup
		},
	}

	// RPC template
	tm.templates["rpc"] = &Template{
		Name:        "rpc",
		Description: "Public RPC node configuration",
		Network:     "custom",
		Values: map[string]interface{}{
			"rpc_apis": []string{"eth", "net", "web3", "txpool"},
			"ws_apis":  []string{"eth", "net", "web3", "txpool"},
			"extra_flags": []string{
				"--rpc.gascap=50000000",
				"--rpc.txfeecap=10",
			},
			"max_peers":          25,
			"health_check_interval": "5s",
			"metrics_enabled":    true,
		},
	}
}

// loadTOMLTemplate loads a TOML template file
func (tm *TemplateManager) loadTOMLTemplate(path string, template *Template) error {
	// This would read and parse a TOML file
	// Implementation depends on file system access
	return fmt.Errorf("not implemented")
}

// loadYAMLTemplate loads a YAML template file
func (tm *TemplateManager) loadYAMLTemplate(path string, template *Template) error {
	// This would read and parse a YAML file
	// Implementation depends on file system access
	return fmt.Errorf("not implemented")
}

// ApplyTemplate applies template values to a configuration
func (tm *TemplateManager) ApplyTemplate(template *Template, wConfig *WemixvisorConfig, nConfig *NodeConfig) error {
	for key, value := range template.Values {
		switch key {
		// Node configuration
		case "network_id":
			if v, ok := value.(uint64); ok {
				nConfig.NetworkID = v
			}
		case "chain_id":
			if v, ok := value.(string); ok {
				nConfig.ChainID = v
			}
		case "rpc_port":
			if v, ok := value.(int); ok {
				nConfig.RPCPort = v
			}
		case "ws_port":
			if v, ok := value.(int); ok {
				nConfig.WSPort = v
			}
		case "p2p_port":
			if v, ok := value.(int); ok {
				nConfig.P2PPort = v
			}
		case "bootnodes":
			if v, ok := value.([]string); ok {
				nConfig.Bootnodes = v
			} else if v, ok := value.([]interface{}); ok {
				nConfig.Bootnodes = make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						nConfig.Bootnodes[i] = s
					}
				}
			}
		case "max_peers":
			if v, ok := value.(int); ok {
				nConfig.MaxPeers = v
			}
		case "validator_mode":
			if v, ok := value.(bool); ok {
				nConfig.ValidatorMode = v
			}
		case "data_dir":
			if v, ok := value.(string); ok {
				nConfig.DataDir = expandPath(v)
			}
		case "extra_flags":
			if v, ok := value.([]string); ok {
				nConfig.ExtraFlags = v
			} else if v, ok := value.([]interface{}); ok {
				nConfig.ExtraFlags = make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						nConfig.ExtraFlags[i] = s
					}
				}
			}

		// Wemixvisor configuration
		case "max_restarts":
			if v, ok := value.(int); ok {
				wConfig.MaxRestarts = v
			}
		case "health_check_interval":
			if v, ok := value.(string); ok {
				if d, err := parseDuration(v); err == nil {
					wConfig.HealthCheckInterval = d
				}
			}
		case "metrics_enabled":
			if v, ok := value.(bool); ok {
				wConfig.MetricsEnabled = v
			}
		case "metrics_interval":
			if v, ok := value.(string); ok {
				if d, err := parseDuration(v); err == nil {
					wConfig.MetricsInterval = d
				}
			}
		case "auto_backup":
			if v, ok := value.(bool); ok {
				wConfig.AutoBackup = v
			}
		case "shutdown_grace":
			if v, ok := value.(string); ok {
				if d, err := parseDuration(v); err == nil {
					wConfig.ShutdownGrace = d
				}
			}

		// RPC/WS API configurations
		case "rpc_apis":
			if v, ok := value.([]string); ok {
				nConfig.RPCAPIs = v
			} else if v, ok := value.([]interface{}); ok {
				nConfig.RPCAPIs = make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						nConfig.RPCAPIs[i] = s
					}
				}
			}
		case "ws_apis":
			if v, ok := value.([]string); ok {
				nConfig.WSAPIs = v
			} else if v, ok := value.([]interface{}); ok {
				nConfig.WSAPIs = make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						nConfig.WSAPIs[i] = s
					}
				}
			}
		}
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := filepath.Join("${HOME}", path[2:])
		return home
	}
	return path
}

// parseDuration parses a duration string
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}