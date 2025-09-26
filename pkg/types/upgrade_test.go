package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseUpgradeInfoFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		want    *UpgradeInfo
	}{
		{
			name: "valid upgrade info",
			content: `{
				"name": "v2.0.0",
				"height": 1000000,
				"info": {
					"binaries": {
						"linux/amd64": "https://example.com/binary"
					}
				}
			}`,
			wantErr: false,
			want: &UpgradeInfo{
				Name:   "v2.0.0",
				Height: 1000000,
				Info: map[string]interface{}{
					"binaries": map[string]interface{}{
						"linux/amd64": "https://example.com/binary",
					},
				},
			},
		},
		{
			name: "empty name",
			content: `{
				"name": "",
				"height": 1000000
			}`,
			wantErr: true,
		},
		{
			name: "zero height",
			content: `{
				"name": "v2.0.0",
				"height": 0
			}`,
			wantErr: true,
		},
		{
			name: "negative height",
			content: `{
				"name": "v2.0.0",
				"height": -1
			}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			content: `{invalid json}`,
			wantErr: true,
		},
		{
			name: "minimal valid info",
			content: `{
				"name": "v1.0.0",
				"height": 1
			}`,
			wantErr: false,
			want: &UpgradeInfo{
				Name:   "v1.0.0",
				Height: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile := filepath.Join(t.TempDir(), "upgrade-info.json")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			// Parse file
			got, err := ParseUpgradeInfoFile(tmpFile)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUpgradeInfoFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check result
			if !tt.wantErr && got != nil {
				if got.Name != tt.want.Name {
					t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Height != tt.want.Height {
					t.Errorf("Height = %v, want %v", got.Height, tt.want.Height)
				}
			}
		})
	}
}

func TestParseUpgradeInfoFileNotExist(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "non-existent.json")
	_, err := ParseUpgradeInfoFile(tmpFile)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWriteUpgradeInfoFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "upgrade-info.json")

	info := &UpgradeInfo{
		Name:   "v3.0.0",
		Height: 2000000,
		Info: map[string]interface{}{
			"binaries": map[string]interface{}{
				"linux/amd64": "https://example.com/v3",
			},
		},
	}

	// Write file
	err := WriteUpgradeInfoFile(tmpFile, info)
	if err != nil {
		t.Fatalf("WriteUpgradeInfoFile() error = %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var readInfo UpgradeInfo
	err = json.Unmarshal(data, &readInfo)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if readInfo.Name != info.Name {
		t.Errorf("Name = %v, want %v", readInfo.Name, info.Name)
	}
	if readInfo.Height != info.Height {
		t.Errorf("Height = %v, want %v", readInfo.Height, info.Height)
	}
}

func TestParseBinaryInfo(t *testing.T) {
	tests := []struct {
		name string
		info map[string]interface{}
		want *BinaryInfo
	}{
		{
			name: "with binaries and checksum",
			info: map[string]interface{}{
				"binaries": map[string]interface{}{
					"linux/amd64":  "https://example.com/linux",
					"darwin/amd64": "https://example.com/darwin",
				},
				"checksum": "sha256:abc123",
			},
			want: &BinaryInfo{
				Binaries: map[string]string{
					"linux/amd64":  "https://example.com/linux",
					"darwin/amd64": "https://example.com/darwin",
				},
				Checksum: "sha256:abc123",
			},
		},
		{
			name: "only binaries",
			info: map[string]interface{}{
				"binaries": map[string]interface{}{
					"any": "https://example.com/any",
				},
			},
			want: &BinaryInfo{
				Binaries: map[string]string{
					"any": "https://example.com/any",
				},
				Checksum: "",
			},
		},
		{
			name: "empty info",
			info: map[string]interface{}{},
			want: &BinaryInfo{
				Binaries: map[string]string{},
				Checksum: "",
			},
		},
		{
			name: "nil info",
			info: nil,
			want: &BinaryInfo{
				Binaries: map[string]string{},
				Checksum: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBinaryInfo(tt.info)
			if err != nil {
				t.Errorf("ParseBinaryInfo() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got.Binaries, tt.want.Binaries) {
				t.Errorf("Binaries = %v, want %v", got.Binaries, tt.want.Binaries)
			}

			if got.Checksum != tt.want.Checksum {
				t.Errorf("Checksum = %v, want %v", got.Checksum, tt.want.Checksum)
			}
		})
	}
}

func TestUpgradePlan(t *testing.T) {
	plan := UpgradePlan{
		Name:   "v4.0.0",
		Height: 3000000,
		Info:   "Major upgrade with new features",
	}

	// Test JSON marshaling
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("failed to marshal plan: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled UpgradePlan
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal plan: %v", err)
	}

	if unmarshaled.Name != plan.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, plan.Name)
	}
	if unmarshaled.Height != plan.Height {
		t.Errorf("Height = %v, want %v", unmarshaled.Height, plan.Height)
	}
	if unmarshaled.Info != plan.Info {
		t.Errorf("Info = %v, want %v", unmarshaled.Info, plan.Info)
	}
}