package atomic

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestInterpolateCommand(t *testing.T) {
	testCases := map[string]struct {
		test           *Test
		arguments      map[string]string
		expectedOutput string
	}{
		"no templating and no arguments": {
			test: &Test{
				Executor:       Executor{Command: aws.String("echo 'Hello World!'")},
				InputArguments: nil,
			},
			arguments:      nil,
			expectedOutput: "echo 'Hello World!'",
		},
		"no templating and some arguments": {
			test: &Test{
				Executor:       Executor{Command: aws.String("echo 'Hello World!'")},
				InputArguments: map[string]InputArgument{"first_name": {}, "last_name": {}},
			},
			arguments:      map[string]string{"first_name": "John", "last_name": "Doe"},
			expectedOutput: "echo 'Hello World!'",
		},
		"templating and no arguments": {
			test: &Test{
				Executor:       Executor{Command: aws.String("echo 'Hello #{first_name} #{last_name}!'")},
				InputArguments: map[string]InputArgument{"first_name": {}, "last_name": {}},
			},
			arguments:      nil,
			expectedOutput: "echo 'Hello  !'",
		},
		"templating and some arguments": {
			test: &Test{
				Executor:       Executor{Command: aws.String("echo 'Hello #{first_name} #{last_name}!'")},
				InputArguments: map[string]InputArgument{"first_name": {}, "last_name": {}},
			},
			arguments:      map[string]string{"first_name": "John"},
			expectedOutput: "echo 'Hello John !'",
		},
		"templating and all arguments": {
			test: &Test{
				Executor:       Executor{Command: aws.String("echo 'Hello #{first_name} #{last_name}!'")},
				InputArguments: map[string]InputArgument{"first_name": {}, "last_name": {}},
			},
			arguments:      map[string]string{"first_name": "John", "last_name": "Doe"},
			expectedOutput: "echo 'Hello John Doe!'",
		},
	}

	for testCaseName, testCaseData := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			assert.Equal(t, testCaseData.expectedOutput, testCaseData.test.interpolateCommand(*testCaseData.test.Executor.Command, testCaseData.arguments))
		})
	}
}

func TestFormatCommandValidation(t *testing.T) {
	testCases := map[string]struct {
		test        *Test
		args        map[string]string
		expectError bool
		errorMsg    string
	}{
		"missing required parameter": {
			test: &Test{
				Executor: Executor{
					Name:    "bash",
					Command: aws.String("echo #{required_param}"),
				},
				InputArguments: map[string]InputArgument{
					"required_param": {}, // No default value
				},
			},
			args:        nil,
			expectError: true,
			errorMsg:    "missing required parameters: required_param",
		},
		"missing multiple required parameters": {
			test: &Test{
				Executor: Executor{
					Name:    "bash",
					Command: aws.String("echo #{param1} #{param2}"),
				},
				InputArguments: map[string]InputArgument{
					"param1": {}, // No default value
					"param2": {}, // No default value
				},
			},
			args:        nil,
			expectError: true,
			errorMsg:    "missing required parameters: param1, param2",
		},
		"optional parameter with default": {
			test: &Test{
				Executor: Executor{
					Name:    "bash",
					Command: aws.String("echo #{optional_param}"),
				},
				InputArguments: map[string]InputArgument{
					"optional_param": {Default: aws.String("default_value")},
				},
			},
			args:        nil,
			expectError: false,
		},
		"required parameter provided": {
			test: &Test{
				Executor: Executor{
					Name:    "bash",
					Command: aws.String("echo #{required_param}"),
				},
				InputArguments: map[string]InputArgument{
					"required_param": {}, // No default value
				},
			},
			args:        map[string]string{"required_param": "provided_value"},
			expectError: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			_, err := testCase.test.FormatCommand(testCase.args)
			if testCase.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatDependencyCommands(t *testing.T) {
	test := &Test{
		Dependencies: []Dependency{
			{
				Description:      "Install curl",
				PreReqCommand:    "which curl",
				GetPreReqCommand: "apt-get install -y curl",
			},
			{
				Description:      "Check network",
				PreReqCommand:    "ping -c 1 #{target}",
				GetPreReqCommand: "",
			},
		},
		InputArguments: map[string]InputArgument{
			"target": {Default: aws.String("google.com")},
		},
	}

	commands := test.FormatDependencyCommands(map[string]string{"target": "example.com"})
	expected := []string{
		"which curl",
		"apt-get install -y curl",
		"ping -c 1 example.com",
	}

	assert.Equal(t, expected, commands)
}

func TestFormatCleanupCommand(t *testing.T) {
	testCases := map[string]struct {
		test     *Test
		args     map[string]string
		expected *string
	}{
		"no cleanup command": {
			test: &Test{
				Executor: Executor{},
			},
			args:     nil,
			expected: nil,
		},
		"cleanup command with interpolation": {
			test: &Test{
				Executor: Executor{
					CleanupCommand: aws.String("rm -f #{file}"),
				},
				InputArguments: map[string]InputArgument{
					"file": {Default: aws.String("/tmp/test.txt")},
				},
			},
			args:     map[string]string{"file": "/tmp/custom.txt"},
			expected: aws.String("rm -f /tmp/custom.txt"),
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := testCase.test.FormatCleanupCommand(testCase.args)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestRequiresElevation(t *testing.T) {
	testCases := map[string]struct {
		test     *Test
		expected bool
	}{
		"no elevation required": {
			test: &Test{
				Executor: Executor{ElevationRequired: aws.Bool(false)},
			},
			expected: false,
		},
		"elevation required": {
			test: &Test{
				Executor: Executor{ElevationRequired: aws.Bool(true)},
			},
			expected: true,
		},
		"elevation not specified": {
			test: &Test{
				Executor: Executor{},
			},
			expected: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := testCase.test.RequiresElevation()
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestFormatCommandExecutorTypes(t *testing.T) {
	testCases := map[string]struct {
		test        *Test
		args        map[string]string
		expected    string
		expectError bool
	}{
		"bash executor": {
			test: &Test{
				Executor: Executor{
					Name:    "bash",
					Command: aws.String("echo hello"),
				},
			},
			args:        nil,
			expected:    "echo hello",
			expectError: false,
		},
		"command_prompt executor": {
			test: &Test{
				Executor: Executor{
					Name:    "command_prompt",
					Command: aws.String("dir C:\\"),
				},
			},
			args:        nil,
			expected:    "ls -la C:/",
			expectError: false,
		},
		"powershell executor": {
			test: &Test{
				Executor: Executor{
					Name:    "powershell",
					Command: aws.String("Get-Process"),
				},
			},
			args:        nil,
			expected:    "ps aux",
			expectError: false,
		},
		"manual executor": {
			test: &Test{
				Executor: Executor{
					Name:    "manual",
					Command: aws.String("Do something manually"),
				},
			},
			args:        nil,
			expected:    "",
			expectError: true,
		},
		"unsupported executor": {
			test: &Test{
				Executor: Executor{
					Name:    "unsupported",
					Command: aws.String("some command"),
				},
			},
			args:        nil,
			expected:    "",
			expectError: true,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result, err := testCase.test.FormatCommand(testCase.args)
			if testCase.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, result)
			}
		})
	}
}
