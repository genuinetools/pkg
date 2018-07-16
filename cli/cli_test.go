package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"testing"
)

const (
	testHelp = `Show the test information.`
)

var (
	nilFunction = func(ctx context.Context) error {
		return nil
	}
	nilActionFunction = func(ctx context.Context, args []string) error {
		return nil
	}

	errExpected            = errors.New("expected error")
	errExpectedFromCommand = errors.New("expected error command error")
	errFunction            = func(ctx context.Context) error {
		return errExpected
	}
)

func (cmd *testCommand) Name() string      { return "test" }
func (cmd *testCommand) Args() string      { return "" }
func (cmd *testCommand) ShortHelp() string { return testHelp }
func (cmd *testCommand) LongHelp() string  { return testHelp }
func (cmd *testCommand) Hidden() bool      { return false }

func (cmd *testCommand) Register(fs *flag.FlagSet) {}

type testCommand struct{}

func (cmd *testCommand) Run(ctx context.Context, args []string) error {
	return nil
}

func (cmd *errorCommand) Name() string      { return "error" }
func (cmd *errorCommand) Args() string      { return "" }
func (cmd *errorCommand) ShortHelp() string { return testHelp }
func (cmd *errorCommand) LongHelp() string  { return testHelp }
func (cmd *errorCommand) Hidden() bool      { return false }

func (cmd *errorCommand) Register(fs *flag.FlagSet) {}

type errorCommand struct{}

func (cmd *errorCommand) Run(ctx context.Context, args []string) error {
	return errExpectedFromCommand
}

func TestProgramWithNoCommandsOrFlagsOrAction(t *testing.T) {
	p := NewProgram()
	testCases := []struct {
		description string
		args        []string
		expectedErr error
	}{
		{
			description: "nil",
		},
		{
			description: "empty",
			args:        []string{},
		},
		{
			description: "args: foo",
			args:        []string{"foo"},
		},
		{
			description: "args: foo bar",
			args:        []string{"foo", "bar"},
			expectedErr: errors.New("bar: no such command"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			printUsage, err := p.run(context.Background(), tc.args)
			compareErrors(t, err, tc.expectedErr)

			if !printUsage {
				t.Fatal("expected behavior was to print the usage")
			}
		})
	}
}

func TestProgramWithNoCommandsOrFlags(t *testing.T) {
	p := NewProgram()
	p.Action = nilActionFunction
	testCases := []struct {
		description      string
		args             []string
		shouldPrintUsage bool
	}{
		{
			description:      "nil",
			shouldPrintUsage: true,
		},
		{
			description:      "empty",
			args:             []string{},
			shouldPrintUsage: true,
		},
		{
			description:      "args: foo",
			args:             []string{"foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo bar",
			args:             []string{"foo", "bar"},
			shouldPrintUsage: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			printUsage, err := p.run(context.Background(), tc.args)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}
}

func TestProgramWithCommandsAndAction(t *testing.T) {
	p := NewProgram()
	p.Commands = []Command{
		&errorCommand{},
		&testCommand{},
	}
	p.Action = nilActionFunction
	testCases := []struct {
		description      string
		args             []string
		shouldPrintUsage bool
		expectedErr      error
	}{
		{
			description:      "nil",
			shouldPrintUsage: true,
		},
		{
			description:      "empty",
			args:             []string{},
			shouldPrintUsage: true,
		},
		{
			description:      "args: foo",
			args:             []string{"foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo bar",
			args:             []string{"foo", "bar"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo test",
			args:             []string{"foo", "test"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo test foo",
			args:             []string{"foo", "test", "foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo test foo bar",
			args:             []string{"foo", "test", "foo", "bar"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo error",
			args:             []string{"foo", "error"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo error foo",
			args:             []string{"foo", "error", "foo"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo error foo bar",
			args:             []string{"foo", "error", "foo", "bar"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo version",
			args:             []string{"foo", "version"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo version foo",
			args:             []string{"foo", "version", "foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo version foo bar",
			args:             []string{"foo", "version", "foo", "bar"},
			shouldPrintUsage: false,
		},
		/*{
			description:      "args: foo version --help",
			args:             []string{"foo", "version", "--help"},
			shouldPrintUsage: true,
		},*/
	}

	// Create the context with the values we need to pass to the version command.
	ctx := context.WithValue(context.Background(), GitCommitKey, p.GitCommit)
	ctx = context.WithValue(ctx, VersionKey, p.Version)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Add a Before.
	p.Before = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with Successful Before -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Add an After.
	p.After = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with successful After -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Test program with an error on After.
	p.After = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on After -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			// When we print the usage for nil and empty, we never hit
			// the After function.
			if !tc.shouldPrintUsage {
				// If we are at the point where the command should fail, we should
				// expect that error.
				if tc.expectedErr == errExpectedFromCommand {
					compareErrors(t, err, errExpectedFromCommand)
				} else {
					compareErrors(t, err, errExpected)
				}
			} else {
				compareErrors(t, err, tc.expectedErr)
			}

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Test program with an error on Before.
	p.Before = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on Before -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			// When we print the usage for nil and empty, we never hit
			// the After function.
			if !tc.shouldPrintUsage {
				compareErrors(t, err, errExpected)
			} else {
				compareErrors(t, err, tc.expectedErr)
			}

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}
}

func TestProgramWithCommands(t *testing.T) {
	p := NewProgram()
	p.Commands = []Command{
		&errorCommand{},
		&testCommand{},
	}
	testCases := []struct {
		description      string
		args             []string
		shouldPrintUsage bool
		expectedErr      error
	}{
		{
			description:      "nil",
			shouldPrintUsage: true,
		},
		{
			description:      "empty",
			args:             []string{},
			shouldPrintUsage: true,
		},
		{
			description:      "args: foo",
			args:             []string{"foo"},
			shouldPrintUsage: true,
		},
		{
			description:      "args: foo bar",
			args:             []string{"foo", "bar"},
			shouldPrintUsage: true,
			expectedErr:      errors.New("bar: no such command"),
		},
		{
			description:      "args: foo test",
			args:             []string{"foo", "test"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo test foo",
			args:             []string{"foo", "test", "foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo test foo bar",
			args:             []string{"foo", "test", "foo", "bar"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo error",
			args:             []string{"foo", "error"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo error foo",
			args:             []string{"foo", "error", "foo"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo error foo bar",
			args:             []string{"foo", "error", "foo", "bar"},
			shouldPrintUsage: false,
			expectedErr:      errExpectedFromCommand,
		},
		{
			description:      "args: foo version",
			args:             []string{"foo", "version"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo version foo",
			args:             []string{"foo", "version", "foo"},
			shouldPrintUsage: false,
		},
		{
			description:      "args: foo version foo bar",
			args:             []string{"foo", "version", "foo", "bar"},
			shouldPrintUsage: false,
		},
	}

	// Create the context with the values we need to pass to the version command.
	ctx := context.WithValue(context.Background(), GitCommitKey, p.GitCommit)
	ctx = context.WithValue(ctx, VersionKey, p.Version)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Add a Before.
	p.Before = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with Successful Before -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Add an After.
	p.After = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with successful After -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			compareErrors(t, err, tc.expectedErr)

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Test program with an error on After.
	p.After = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on After -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(ctx, tc.args)
			// When we print the usage for nil and empty, we never hit
			// the After function.
			if !tc.shouldPrintUsage {
				// If we are at the point where the command should fail, we should
				// expect that error.
				if tc.expectedErr == errExpectedFromCommand {
					compareErrors(t, err, errExpectedFromCommand)
				} else {
					compareErrors(t, err, errExpected)
				}
			} else {
				compareErrors(t, err, tc.expectedErr)
			}

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}

	// Test program with an error on Before.
	p.Before = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on Before -> %s", tc.description), func(t *testing.T) {
			printUsage, err := p.run(context.Background(), tc.args)
			// When we print the usage for nil and empty, we never hit
			// the After function.
			if !tc.shouldPrintUsage {
				compareErrors(t, err, errExpected)
			} else {
				compareErrors(t, err, tc.expectedErr)
			}

			if printUsage != tc.shouldPrintUsage {
				t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
			}
		})
	}
}

func compareErrors(t *testing.T, err, expectedErr error) {
	if expectedErr != nil {
		if err == nil || err.Error() != expectedErr.Error() {
			t.Fatalf("expected error %#v got: %#v", expectedErr, err)
		}

		return
	}

	if err != expectedErr {
		t.Fatalf("expected error %#v got: %#v", expectedErr, err)
	}

	return
}
