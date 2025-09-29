package cli

import (
	"reflect"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *ParsedArgs
		wantErr bool
	}{
		{
			name: "start with geth args",
			args: []string{"start", "--datadir", "/data", "--syncmode", "full", "--home", "/home/wemix"},
			want: &ParsedArgs{
				Command:        "start",
				WemixvisorOpts: map[string]string{"--home": "/home/wemix"},
				NodeArgs:       []string{"--datadir", "/data", "--syncmode", "full"},
			},
			wantErr: false,
		},
		{
			name: "stop command",
			args: []string{"stop"},
			want: &ParsedArgs{
				Command:        "stop",
				WemixvisorOpts: map[string]string{},
				NodeArgs:       []string{},
			},
			wantErr: false,
		},
		{
			name: "restart command",
			args: []string{"restart", "--home", "/custom"},
			want: &ParsedArgs{
				Command:        "restart",
				WemixvisorOpts: map[string]string{"--home": "/custom"},
				NodeArgs:       []string{},
			},
			wantErr: false,
		},
		{
			name: "status with json flag",
			args: []string{"status", "--json"},
			want: &ParsedArgs{
				Command:        "status",
				WemixvisorOpts: map[string]string{"--json": "true"},
				NodeArgs:       []string{},
			},
			wantErr: false,
		},
		{
			name: "start with mixed args",
			args: []string{"start", "--debug", "--datadir=/data", "--network=testnet", "--port", "30303"},
			want: &ParsedArgs{
				Command:        "start",
				WemixvisorOpts: map[string]string{"--debug": "true", "--network": "testnet"},
				NodeArgs:       []string{"--datadir=/data", "--port", "30303"},
			},
			wantErr: false,
		},
		{
			name:    "no command",
			args:    []string{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid command",
			args:    []string{"invalid"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "stop with unexpected args",
			args:    []string{"stop", "--datadir", "/data"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ExtractGethCompatibleArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "mixed args",
			args: []string{"start", "--home", "/home", "--datadir", "/data", "--debug"},
			want: []string{"--datadir", "/data"},
		},
		{
			name: "only geth args",
			args: []string{"--datadir", "/data", "--syncmode", "full", "--port", "30303"},
			want: []string{"--datadir", "/data", "--syncmode", "full", "--port", "30303"},
		},
		{
			name: "only wemixvisor args",
			args: []string{"start", "--home", "/home", "--debug", "--json"},
			want: nil,
		},
		{
			name: "empty args",
			args: []string{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got := p.ExtractGethCompatibleArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractGethCompatibleArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildNodeArgs(t *testing.T) {
	tests := []struct {
		name     string
		parsed   *ParsedArgs
		defaults map[string]string
		want     []string
	}{
		{
			name: "with defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data"},
			},
			defaults: map[string]string{
				"--syncmode": "full",
				"--port":     "30303",
			},
			want: []string{"--datadir", "/data", "--port", "30303", "--syncmode", "full"},
		},
		{
			name: "override defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data", "--syncmode", "light"},
			},
			defaults: map[string]string{
				"--syncmode": "full",
				"--port":     "30303",
			},
			want: []string{"--datadir", "/data", "--syncmode", "light", "--port", "30303"},
		},
		{
			name: "no defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data"},
			},
			defaults: map[string]string{},
			want:     []string{"--datadir", "/data"},
		},
		{
			name: "empty node args",
			parsed: &ParsedArgs{
				NodeArgs: []string{},
			},
			defaults: map[string]string{
				"--syncmode": "full",
			},
			want: []string{"--syncmode", "full"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNodeArgs(tt.parsed, tt.defaults)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildNodeArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"start", true},
		{"stop", true},
		{"restart", true},
		{"status", true},
		{"logs", true},
		{"version", true},
		{"init", true},
		{"run", true},
		{"invalid", false},
		{"", false},
		{"START", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := isValidCommand(tt.cmd); got != tt.want {
				t.Errorf("isValidCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestContainsFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{
			name: "flag present",
			args: []string{"--datadir", "/data", "--syncmode", "full"},
			flag: "--datadir",
			want: true,
		},
		{
			name: "flag with equals",
			args: []string{"--datadir=/data", "--syncmode=full"},
			flag: "--datadir",
			want: true,
		},
		{
			name: "flag not present",
			args: []string{"--syncmode", "full"},
			flag: "--datadir",
			want: false,
		},
		{
			name: "empty args",
			args: []string{},
			flag: "--datadir",
			want: false,
		},
		{
			name: "partial match should not count",
			args: []string{"--data", "/data"},
			flag: "--datadir",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsFlag(tt.args, tt.flag); got != tt.want {
				t.Errorf("containsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}