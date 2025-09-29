package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Parser handles CLI argument parsing and validation
type Parser struct {
	// Wemixvisor-specific flags
	wemixvisorFlags map[string]bool
	// Geth pass-through arguments
	nodeArgs []string
}

// NewParser creates a new CLI parser
func NewParser() *Parser {
	return &Parser{
		wemixvisorFlags: map[string]bool{
			"--home":    true,
			"--name":    true,
			"--network": true,
			"--debug":   true,
			"--json":    true,
			"--quiet":   true,
		},
		nodeArgs: []string{},
	}
}

// ParsedArgs contains the parsed command line arguments
type ParsedArgs struct {
	Command       string            // start, stop, restart, status, logs, etc.
	WemixvisorOpts map[string]string // Wemixvisor-specific options
	NodeArgs      []string          // Pass-through arguments for geth
}

// Parse parses the command line arguments
func (p *Parser) Parse(args []string) (*ParsedArgs, error) {
	if len(args) == 0 {
		return nil, errors.New("no command specified")
	}

	parsed := &ParsedArgs{
		Command:        args[0],
		WemixvisorOpts: make(map[string]string),
		NodeArgs:       []string{},
	}

	// Validate command
	if !isValidCommand(parsed.Command) {
		return nil, fmt.Errorf("unknown command: %s", parsed.Command)
	}

	// Parse remaining arguments
	i := 1
	for i < len(args) {
		arg := args[i]

		// Check if it's a wemixvisor flag
		if p.isWemixvisorFlag(arg) {
			// Extract flag name and value
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				parsed.WemixvisorOpts[parts[0]] = parts[1]
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Next arg is the value
				parsed.WemixvisorOpts[arg] = args[i+1]
				i++ // Skip the value
			} else {
				// Boolean flag
				parsed.WemixvisorOpts[arg] = "true"
			}
		} else {
			// Pass through to node
			parsed.NodeArgs = append(parsed.NodeArgs, arg)
		}
		i++
	}

	// Validate parsed arguments
	if err := p.validate(parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

// isWemixvisorFlag checks if a flag is specific to wemixvisor
func (p *Parser) isWemixvisorFlag(arg string) bool {
	// Remove '=' and everything after for checking
	flag := arg
	if idx := strings.Index(arg, "="); idx != -1 {
		flag = arg[:idx]
	}

	_, exists := p.wemixvisorFlags[flag]
	return exists
}

// validate validates the parsed arguments
func (p *Parser) validate(args *ParsedArgs) error {
	// Command-specific validation
	switch args.Command {
	case "start":
		// Start can have node arguments
		if len(args.NodeArgs) == 0 {
			// It's okay to start with default arguments
		}
	case "stop", "status":
		// These commands shouldn't have node arguments
		if len(args.NodeArgs) > 0 {
			return fmt.Errorf("%s command does not accept node arguments", args.Command)
		}
	case "restart":
		// Restart can optionally have new node arguments
	case "logs":
		// Logs command can have options like --follow, --tail
		// These are handled separately
	case "version":
		// No validation needed
	default:
		// Unknown commands should have been caught earlier
		return fmt.Errorf("invalid command: %s", args.Command)
	}

	return nil
}

// isValidCommand checks if a command is valid
func isValidCommand(cmd string) bool {
	validCommands := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"status":  true,
		"logs":    true,
		"version": true,
		"init":    true,
		"run":     true,
	}

	_, exists := validCommands[cmd]
	return exists
}

// ExtractGethCompatibleArgs processes arguments for geth compatibility
func (p *Parser) ExtractGethCompatibleArgs(args []string) []string {
	var gethArgs []string

	for i := 0; i < len(args); {
		arg := args[i]

		// Skip wemixvisor-specific commands
		if isValidCommand(arg) {
			i++
			continue
		}

		// Check if it's a wemixvisor-specific flag
		if p.isWemixvisorFlag(arg) {
			// Skip the flag
			i++
			// If the flag has a separate value (not with =), skip the value too
			if !strings.Contains(arg, "=") && i < len(args) && !strings.HasPrefix(args[i], "-") {
				i++ // Skip the value
			}
		} else {
			// It's a geth argument, keep it
			gethArgs = append(gethArgs, arg)
			i++
		}
	}

	return gethArgs
}

// BuildNodeArgs builds the final arguments to pass to the node
func BuildNodeArgs(parsed *ParsedArgs, defaults map[string]string) []string {
	var args []string

	// Start with parsed node arguments
	args = append(args, parsed.NodeArgs...)

	// Add defaults in a deterministic order
	// First collect and sort the keys
	var keys []string
	for key := range defaults {
		if !containsFlag(parsed.NodeArgs, key) {
			keys = append(keys, key)
		}
	}

	// Sort keys for deterministic order
	sort.Strings(keys)

	// Add defaults in sorted order
	for _, key := range keys {
		value := defaults[key]
		if value != "" {
			args = append(args, key, value)
		} else {
			args = append(args, key)
		}
	}

	return args
}

// containsFlag checks if a flag is already in the arguments
func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}