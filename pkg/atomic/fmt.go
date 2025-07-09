package atomic

import (
	"fmt"
	"strings"
)

func (t *Test) FormatCommand(arguments map[string]string) (string, error) {
	// Validate executor type
	switch t.Executor.Name {
	case "bash", "sh":
		// Unix shell executors - supported
	//case "command_prompt", "cmd":
		// Windows command prompt - implement support
	//case "powershell", "powershell.exe":
		// PowerShell - implement support
	default:
		return "", fmt.Errorf("unsupported executor type `%s`. Supported types: bash, sh, command_prompt, cmd, powershell", t.Executor.Name)
	}

	if t.Executor.Command == nil {
		return "", fmt.Errorf("no command was specified for this test")
	}

	// Validate required parameters
	if err := t.validateRequiredParameters(arguments); err != nil {
		return "", err
	}

	command := t.interpolateCommand(*t.Executor.Command, arguments)

	return command, nil
}

// FormatDependencyCommands returns commands needed to install dependencies
func (t *Test) FormatDependencyCommands(arguments map[string]string) []string {
	var commands []string
	for _, dependency := range t.Dependencies {
		if dependency.PreReqCommand != "" {
			commands = append(commands, t.interpolateCommand(dependency.PreReqCommand, arguments))
		}
		if dependency.GetPreReqCommand != "" {
			commands = append(commands, t.interpolateCommand(dependency.GetPreReqCommand, arguments))
		}
	}
	return commands
}

// FormatCleanupCommand returns the cleanup command if available
func (t *Test) FormatCleanupCommand(arguments map[string]string) *string {
	if t.Executor.CleanupCommand == nil {
		return nil
	}
	cleanupCmd := t.interpolateCommand(*t.Executor.CleanupCommand, arguments)
	return &cleanupCmd
}

// GetSupportedPlatforms returns the platforms this test supports
func (t *Test) GetSupportedPlatforms() []string {
	return t.SupportedPlatforms
}

// RequiresElevation returns true if the test requires elevated privileges
func (t *Test) RequiresElevation() bool {
	return t.Executor.ElevationRequired != nil && *t.Executor.ElevationRequired
}

func (t *Test) interpolateCommand(command string, arguments map[string]string) string {
	for parameterName, parameterDefinition := range t.InputArguments {
		var parameterValue string

		if _, ok := arguments[parameterName]; ok {
			parameterValue = arguments[parameterName]
		} else if parameterDefinition.Default != nil {
			parameterValue = *parameterDefinition.Default
		}

		command = strings.ReplaceAll(command, fmt.Sprintf("#{%s}", parameterName), parameterValue)
	}

	return command
}

// validateRequiredParameters checks that all required parameters (those without defaults) are provided
func (t *Test) validateRequiredParameters(arguments map[string]string) error {
	var missingParams []string
	
	for parameterName, parameterDefinition := range t.InputArguments {
		// If parameter has no default value and is not provided in arguments, it's required
		if parameterDefinition.Default == nil {
			if _, ok := arguments[parameterName]; !ok {
				missingParams = append(missingParams, parameterName)
			}
		}
	}
	
	if len(missingParams) > 0 {
		return fmt.Errorf("missing required parameters: %s", strings.Join(missingParams, ", "))
	}
	
	return nil
}
