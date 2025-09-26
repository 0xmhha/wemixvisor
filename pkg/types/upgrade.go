package types

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// UpgradeInfo represents upgrade information
type UpgradeInfo struct {
	Name   string                 `json:"name"`
	Height int64                  `json:"height"`
	Info   map[string]interface{} `json:"info,omitempty"`
}

// UpgradePlan represents a planned upgrade
type UpgradePlan struct {
	Name   string    `json:"name"`
	Height int64     `json:"height"`
	Time   time.Time `json:"time,omitempty"`
	Info   string    `json:"info,omitempty"`
}

// ParseUpgradeInfoFile parses the upgrade-info.json file
func ParseUpgradeInfoFile(filename string) (*UpgradeInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read upgrade info file: %w", err)
	}

	var info UpgradeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse upgrade info: %w", err)
	}

	if info.Name == "" {
		return nil, fmt.Errorf("upgrade name is empty")
	}

	if info.Height <= 0 {
		return nil, fmt.Errorf("upgrade height must be positive")
	}

	return &info, nil
}

// WriteUpgradeInfoFile writes upgrade info to file
func WriteUpgradeInfoFile(filename string, info *UpgradeInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal upgrade info: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write upgrade info file: %w", err)
	}

	return nil
}

// BinaryInfo contains information about binaries for different platforms
type BinaryInfo struct {
	Binaries map[string]string `json:"binaries"`
	Checksum string            `json:"checksum,omitempty"`
}

// ParseBinaryInfo extracts binary info from upgrade info
func ParseBinaryInfo(info map[string]interface{}) (*BinaryInfo, error) {
	binInfo := &BinaryInfo{
		Binaries: make(map[string]string),
	}

	if binaries, ok := info["binaries"].(map[string]interface{}); ok {
		for platform, url := range binaries {
			if urlStr, ok := url.(string); ok {
				binInfo.Binaries[platform] = urlStr
			}
		}
	}

	if checksum, ok := info["checksum"].(string); ok {
		binInfo.Checksum = checksum
	}

	return binInfo, nil
}