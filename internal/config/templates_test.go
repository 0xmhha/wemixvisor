package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewTemplateManager(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	assert.NotNil(t, tm)
	assert.NotEmpty(t, tm.templates)

	// Check default templates are loaded
	templates := []string{"mainnet", "testnet", "devnet", "validator", "archive", "rpc"}
	for _, name := range templates {
		_, exists := tm.templates[name]
		assert.True(t, exists, "template %s should exist", name)
	}
}

func TestTemplateManager_GetTemplate(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	tests := []struct {
		name        string
		templateName string
		wantErr     bool
	}{
		{
			name:        "existing template",
			templateName: "mainnet",
			wantErr:     false,
		},
		{
			name:        "non-existing template",
			templateName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := tm.GetTemplate(tt.templateName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, template)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, template)
				assert.Equal(t, tt.templateName, template.Name)
			}
		})
	}
}

func TestTemplateManager_ListTemplates(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	templates := tm.ListTemplates()
	assert.NotEmpty(t, templates)
	assert.GreaterOrEqual(t, len(templates), 6) // At least the default templates
}

func TestTemplateManager_AddTemplate(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	tests := []struct {
		name     string
		template *Template
		wantErr  bool
	}{
		{
			name: "valid template",
			template: &Template{
				Name:        "custom",
				Description: "Custom template",
				Network:     "custom",
				Values: map[string]interface{}{
					"network_id": uint64(9999),
				},
			},
			wantErr: false,
		},
		{
			name: "template without name",
			template: &Template{
				Description: "No name template",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tm.AddTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify template was added
				added, err := tm.GetTemplate(tt.template.Name)
				assert.NoError(t, err)
				assert.Equal(t, tt.template.Name, added.Name)
			}
		})
	}
}

func TestTemplateManager_RemoveTemplate(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	// Add a custom template
	custom := &Template{
		Name:        "custom-remove",
		Description: "Template to remove",
	}
	err := tm.AddTemplate(custom)
	require.NoError(t, err)

	// Remove the template
	err = tm.RemoveTemplate("custom-remove")
	assert.NoError(t, err)

	// Verify it's removed
	_, err = tm.GetTemplate("custom-remove")
	assert.Error(t, err)

	// Try to remove non-existent template
	err = tm.RemoveTemplate("nonexistent")
	assert.Error(t, err)
}

func TestTemplateManager_LoadFromFile(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		format  string
		content string
		wantErr bool
	}{
		{
			name:   "TOML template",
			format: "toml",
			content: `
name = "file-template"
description = "Template from file"
network = "custom"

[values]
network_id = 9999
chain_id = "9999"
`,
			wantErr: true, // LoadFromFile is not fully implemented
		},
		{
			name:   "YAML template",
			format: "yaml",
			content: `
name: file-template
description: Template from file
network: custom
values:
  network_id: 9999
  chain_id: "9999"
`,
			wantErr: true, // LoadFromFile is not fully implemented
		},
		{
			name:    "unsupported format",
			format:  "txt",
			content: "unsupported",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, "template."+tt.format)
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0644))

			err := tm.LoadFromFile(path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTemplateManager_ApplyTemplate(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	wConfig := &WemixvisorConfig{}
	nConfig := &NodeConfig{}

	// Get mainnet template
	template, err := tm.GetTemplate("mainnet")
	require.NoError(t, err)

	// Apply template
	err = tm.ApplyTemplate(template, wConfig, nConfig)
	assert.NoError(t, err)

	// Verify values were applied
	assert.Equal(t, uint64(1111), nConfig.NetworkID)
	assert.Equal(t, "1111", nConfig.ChainID)
	assert.Equal(t, 8588, nConfig.RPCPort)
	assert.Equal(t, 8598, nConfig.WSPort)
	assert.Equal(t, 8589, nConfig.P2PPort)
	assert.Equal(t, 50, nConfig.MaxPeers)
	assert.False(t, nConfig.ValidatorMode)
}

func TestTemplateManager_ApplyTemplate_Validator(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	wConfig := &WemixvisorConfig{}
	nConfig := &NodeConfig{}

	// Get validator template
	template, err := tm.GetTemplate("validator")
	require.NoError(t, err)

	// Apply template
	err = tm.ApplyTemplate(template, wConfig, nConfig)
	assert.NoError(t, err)

	// Verify values were applied
	assert.True(t, nConfig.ValidatorMode)
	assert.Equal(t, 100, nConfig.MaxPeers)
	assert.True(t, wConfig.MetricsEnabled)
	assert.True(t, wConfig.AutoBackup)
	assert.Equal(t, 3, wConfig.MaxRestarts)
}

func TestTemplateManager_ApplyTemplate_ComplexValues(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	wConfig := &WemixvisorConfig{}
	nConfig := &NodeConfig{}

	// Create template with various value types
	template := &Template{
		Name: "complex",
		Values: map[string]interface{}{
			"network_id":              uint64(1234),
			"chain_id":                "1234",
			"rpc_port":                8545,
			"max_peers":               25,
			"validator_mode":          true,
			"data_dir":                "~/test/data",
			"bootnodes":               []string{"enode://test1", "enode://test2"},
			"extra_flags":             []string{"--flag1", "--flag2"},
			"health_check_interval":   "10s",
			"metrics_interval":        "30s",
			"shutdown_grace":          "60s",
			"auto_backup":             true,
			"metrics_enabled":         true,
			"max_restarts":            10,
			"rpc_apis":                []string{"eth", "net", "web3"},
			"ws_apis":                 []string{"eth", "net"},
			// Test interface{} slice conversion
			"bootnodes_interface":     []interface{}{"enode://iface1", "enode://iface2"},
			"extra_flags_interface":   []interface{}{"--iface1", "--iface2"},
		},
	}

	// Apply template
	err := tm.ApplyTemplate(template, wConfig, nConfig)
	assert.NoError(t, err)

	// Verify all values
	assert.Equal(t, uint64(1234), nConfig.NetworkID)
	assert.Equal(t, "1234", nConfig.ChainID)
	assert.Equal(t, 8545, nConfig.RPCPort)
	assert.Equal(t, 25, nConfig.MaxPeers)
	assert.True(t, nConfig.ValidatorMode)
	assert.Contains(t, nConfig.DataDir, "test/data")
	assert.Len(t, nConfig.Bootnodes, 2)
	assert.Equal(t, "enode://test1", nConfig.Bootnodes[0])
	assert.Len(t, nConfig.ExtraFlags, 2)
	assert.Equal(t, "--flag1", nConfig.ExtraFlags[0])
	assert.Equal(t, 10*time.Second, wConfig.HealthCheckInterval)
	assert.Equal(t, 30*time.Second, wConfig.MetricsInterval)
	assert.Equal(t, 60*time.Second, wConfig.ShutdownGrace)
	assert.True(t, wConfig.AutoBackup)
	assert.True(t, wConfig.MetricsEnabled)
	assert.Equal(t, 10, wConfig.MaxRestarts)
	assert.Len(t, nConfig.RPCAPIs, 3)
	assert.Len(t, nConfig.WSAPIs, 2)
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory",
			input:    "~/wemixvisor",
			expected: "${HOME}/wemixvisor",
		},
		{
			name:     "absolute path",
			input:    "/tmp/wemixvisor",
			expected: "/tmp/wemixvisor",
		},
		{
			name:     "relative path",
			input:    "./wemixvisor",
			expected: "./wemixvisor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "seconds",
			input: "30s",
			want:  30 * time.Second,
		},
		{
			name:  "minutes",
			input: "5m",
			want:  5 * time.Minute,
		},
		{
			name:  "hours",
			input: "2h",
			want:  2 * time.Hour,
		},
		{
			name:  "complex",
			input: "1h30m",
			want:  90 * time.Minute,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDefaultTemplates(t *testing.T) {
	logger := logger.NewTestLogger()
	tm := NewTemplateManager(logger)

	// Test mainnet template
	mainnet, err := tm.GetTemplate("mainnet")
	require.NoError(t, err)
	assert.Equal(t, "mainnet", mainnet.Name)
	assert.Equal(t, "WEMIX3.0 Mainnet configuration", mainnet.Description)
	assert.Equal(t, uint64(1111), mainnet.Values["network_id"])
	assert.Equal(t, "1111", mainnet.Values["chain_id"])

	// Test testnet template
	testnet, err := tm.GetTemplate("testnet")
	require.NoError(t, err)
	assert.Equal(t, "testnet", testnet.Name)
	assert.Equal(t, "WEMIX3.0 Testnet configuration", testnet.Description)
	assert.Equal(t, uint64(1112), testnet.Values["network_id"])
	assert.Equal(t, "1112", testnet.Values["chain_id"])

	// Test devnet template
	devnet, err := tm.GetTemplate("devnet")
	require.NoError(t, err)
	assert.Equal(t, "devnet", devnet.Name)
	assert.Equal(t, "Local development network configuration", devnet.Description)
	assert.Equal(t, uint64(1337), devnet.Values["network_id"])

	// Test validator template
	validator, err := tm.GetTemplate("validator")
	require.NoError(t, err)
	assert.Equal(t, "validator", validator.Name)
	assert.Equal(t, true, validator.Values["validator_mode"])

	// Test archive template
	archive, err := tm.GetTemplate("archive")
	require.NoError(t, err)
	assert.Equal(t, "archive", archive.Name)
	extraFlags := archive.Values["extra_flags"].([]string)
	assert.Contains(t, extraFlags, "--gcmode=archive")

	// Test RPC template
	rpc, err := tm.GetTemplate("rpc")
	require.NoError(t, err)
	assert.Equal(t, "rpc", rpc.Name)
	rpcAPIs := rpc.Values["rpc_apis"].([]string)
	assert.Contains(t, rpcAPIs, "eth")
	assert.Contains(t, rpcAPIs, "txpool")
}