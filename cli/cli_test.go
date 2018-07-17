package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
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

	versionExpected = "ship:\n version"
)

type testCase struct {
	description        string
	args               []string
	shouldPrintUsage   bool
	shouldPrintVersion bool
	expectedErr        error
}

// Define the testCommand.
type testCommand struct{}

func (cmd *testCommand) Name() string                                 { return "test" }
func (cmd *testCommand) Args() string                                 { return "" }
func (cmd *testCommand) ShortHelp() string                            { return testHelp }
func (cmd *testCommand) LongHelp() string                             { return testHelp }
func (cmd *testCommand) Hidden() bool                                 { return false }
func (cmd *testCommand) Register(fs *flag.FlagSet)                    {}
func (cmd *testCommand) Run(ctx context.Context, args []string) error { return nil }

// Define the errorCommand.
type errorCommand struct{}

func (cmd *errorCommand) Name() string                                 { return "error" }
func (cmd *errorCommand) Args() string                                 { return "" }
func (cmd *errorCommand) ShortHelp() string                            { return testHelp }
func (cmd *errorCommand) LongHelp() string                             { return testHelp }
func (cmd *errorCommand) Hidden() bool                                 { return false }
func (cmd *errorCommand) Register(fs *flag.FlagSet)                    {}
func (cmd *errorCommand) Run(ctx context.Context, args []string) error { return errExpectedFromCommand }

func testCasesEmpty() []testCase {
	return []testCase{
		{
			description:      "nil",
			shouldPrintUsage: true,
		},
		{
			description:      "empty",
			args:             []string{},
			shouldPrintUsage: true,
		},
	}
}

func testCasesUndefinedCommand() []testCase {
	return []testCase{
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
	}
}

func testCasesWithCommands() []testCase {
	return []testCase{
		{
			description: "args: foo test",
			args:        []string{"foo", "test"},
		},
		{
			description: "args: foo test foo",
			args:        []string{"foo", "test", "foo"},
		},
		{
			description: "args: foo test foo bar",
			args:        []string{"foo", "test", "foo", "bar"},
		},
		{
			description: "args: foo error",
			args:        []string{"foo", "error"},
			expectedErr: errExpectedFromCommand,
		},
		{
			description: "args: foo error foo",
			args:        []string{"foo", "error", "foo"},
			expectedErr: errExpectedFromCommand,
		},
		{
			description: "args: foo error foo bar",
			args:        []string{"foo", "error", "foo", "bar"},
			expectedErr: errExpectedFromCommand,
		},
		{
			description:        "args: foo version",
			args:               []string{"foo", "version"},
			shouldPrintVersion: true,
		},
		{
			description:        "args: foo version foo",
			args:               []string{"foo", "version", "foo"},
			shouldPrintVersion: true,
		},
		{
			description:        "args: foo version foo bar",
			args:               []string{"foo", "version", "foo", "bar"},
			shouldPrintVersion: true,
		},
	}
}

func testCasesHelp() []testCase {
	return []testCase{
		{
			description: "args: foo --help",
			args:        []string{"foo", "--help"},
		},
		{
			description: "args: foo help",
			args:        []string{"foo", "help"},
		},
		{
			description: "args: foo help bar --thing",
			args:        []string{"foo", "help", "bar", "--thing"},
		},
		{
			description: "args: foo bar --help",
			args:        []string{"foo", "bar", "--help"},
		},
		{
			description: "args: foo test --help",
			args:        []string{"foo", "test", "--help"},
		},
		{
			description: "args: foo -h test foo",
			args:        []string{"foo", "-h", "test", "foo", "--help"},
		},
		{
			description: "args: foo error -h",
			args:        []string{"foo", "error", "-h"},
		},
		{
			description: "args: foo error foo --help",
			args:        []string{"foo", "error", "foo", "--help"},
		},
		{
			description: "args: foo error foo bar --help",
			args:        []string{"foo", "error", "foo", "bar", "--help"},
		},
		{
			description: "args: foo version foo --help",
			args:        []string{"foo", "version", "foo", "--help"},
		},
		{
			description: "args: foo version foo bar -h",
			args:        []string{"foo", "version", "foo", "bar", "-h"},
		},
	}
}

func testCasesWithAction() []testCase {
	return []testCase{
		{
			description: "args: foo",
			args:        []string{"foo"},
		},
		{
			description: "args: foo bar",
			args:        []string{"foo", "bar"},
		},
	}
}

func TestProgramUsage(t *testing.T) {
	var (
		debug  bool
		token  string
		output string

		expectedOutput = `sample -  My sample command line tool.

Usage: sample <command>

Flags:

  -d, --debug  enable debug logging (default: false)
  -o           where to save the output (default: defaultOutput)
  -t, --thing  a flag for thing (default: false)
  --token      API token (default: <none>)

Commands:

  error    Show the test information.
  test     Show the test information.
  version  Show the version information.

`

		expectedVersionOutput = `Usage: sample version` + " " + `

Show the version information.

Flags:

  -d, --debug  enable debug logging (default: false)
  -o           where to save the output (default: defaultOutput)
  -t, --thing  a flag for thing (default: false)
  --token      API token (default: <none>)

`
	)

	// Setup the program.
	p := NewProgram()
	p.Name = "sample"
	p.Description = "My sample command line tool"

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.StringVar(&token, "token", "", "API token")
	p.FlagSet.StringVar(&output, "o", "defaultOutput", "where to save the output")
	p.FlagSet.BoolVar(&debug, "thing", false, "a flag for thing")
	p.FlagSet.BoolVar(&debug, "t", false, "a flag for thing")
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")
	p.FlagSet.BoolVar(&debug, "debug", false, "enable debug logging")

	p.Commands = []Command{
		&errorCommand{},
		&testCommand{},
	}
	p.Action = nilActionFunction

	p.Run()

	c := startCapture(t)
	if err := p.usage(p.defaultContext()); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := c.finish()
	if stderr != expectedOutput {
		t.Fatalf("expected: %s\ngot: %s", expectedOutput, stderr)
	}
	if len(stdout) > 0 {
		t.Fatalf("expected no stdout, got: %s", stdout)
	}

	// Test versionCommand.
	vcmd := &versionCommand{}
	c = startCapture(t)
	p.resetCommandUsage(vcmd)
	p.FlagSet.Usage()
	stdout, stderr = c.finish()
	if stderr != expectedVersionOutput {
		t.Fatalf("expected: %q\ngot: %q", expectedVersionOutput, stderr)
	}
	if len(stdout) > 0 {
		t.Fatalf("expected no stdout, got: %s", stdout)
	}
}

func TestProgramWithNoCommandsOrFlagsOrAction(t *testing.T) {
	p := NewProgram()
	testCases := append(testCasesEmpty(), testCasesUndefinedCommand()...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}
}

func TestProgramWithNoCommandsOrFlags(t *testing.T) {
	p := NewProgram()
	p.Action = nilActionFunction
	testCases := append(testCasesEmpty(), testCasesWithAction()...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}
}

func TestProgramHelpFlag(t *testing.T) {
	p := NewProgram()
	testCases := testCasesHelp()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c := startCapture(t)
			printUsage, err := p.run(p.defaultContext(), tc.args)
			stdout, stderr := c.finish()
			if strings.Contains(stdout, versionExpected) {
				t.Fatalf("did not expect version information to print, got %s", stdout)
			}
			if err != nil {
				t.Fatalf("expected no error from run, got %v", err)
			}
			if !printUsage {
				t.Fatal("expected printUsage to be true")
			}
			if len(stderr) > 0 {
				t.Fatalf("expected no stderr, got: %s", stderr)
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
	testCases := append(append(testCasesEmpty(),
		testCasesWithCommands()...),
		testCasesWithAction()...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Add a Before.
	p.Before = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with Successful Before -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Add an After.
	p.After = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with successful After -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Test program with an error on After.
	p.After = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on After -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Test program with an error on Before.
	p.Before = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on Before -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}
}

func TestProgramWithCommands(t *testing.T) {
	p := NewProgram()
	p.Commands = []Command{
		&errorCommand{},
		&testCommand{},
	}
	testCases := append(append(testCasesEmpty(),
		testCasesUndefinedCommand()...),
		testCasesWithCommands()...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Add a Before.
	p.Before = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with Successful Before -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Add an After.
	p.After = nilFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with successful After -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Test program with an error on After.
	p.After = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on After -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
		})
	}

	// Test program with an error on Before.
	p.Before = errFunction
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with error on Before -> %s", tc.description), func(t *testing.T) {
			p.doTestRun(t, tc)
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

type capture struct {
	stdout, stderr *os.File
	ro, re         *os.File
	wo, we         *os.File
	co, ce         chan string
}

func startCapture(t *testing.T) capture {
	c := capture{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	// Pipe it to a reader and writer.
	var err error
	c.ro, c.wo, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = c.wo
	c.re, c.we, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = c.we

	return c
}

func (c *capture) finish() (string, string) {
	defer c.ro.Close()
	defer c.re.Close()

	// Copy the output in a separate goroutine so printing can't block indefinitely.
	c.co = make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, c.ro)
		c.co <- buf.String()
	}()
	c.ce = make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, c.re)
		c.ce <- buf.String()
	}()

	// Close everything.
	c.wo.Close()
	c.we.Close()

	// Reset.
	os.Stdout = c.stdout
	os.Stderr = c.stderr

	o := <-c.co
	e := <-c.ce
	return o, e
}

func (p *Program) isErrorOnBefore() bool {
	return p.Before != nil && p.Before(context.Background()) != nil
}

func (p *Program) isErrorOnAfter() bool {
	return p.After != nil && p.After(context.Background()) != nil
}

func (p *Program) doTestRun(t *testing.T, tc testCase) {
	c := startCapture(t)
	printUsage, err := p.run(p.defaultContext(), tc.args)
	stdout, stderr := c.finish()
	if len(stderr) > 0 {
		t.Fatalf("expected no stderr, got: %s", stderr)
	}

	// IF
	// we DON'T EXPECT and error on Before
	// AND
	// we EXPECT the version to be printed
	// THEN
	// check that the version was actually printed.
	if !p.isErrorOnBefore() &&
		tc.shouldPrintVersion && !strings.HasPrefix(stdout, versionExpected) {
		t.Fatalf("expected output to start with %q, got %q", versionExpected, stdout)
	}

	// IF
	// we DON'T EXPECT an error on Before OR After
	// OR
	// we EXPECT the usage to be printed (<nil> or empty)
	// OR
	// we DON'T EXPECT an error on Before but we EXPECT an error on After AND the command was EXPECTED to error
	// THEN
	// check we got the expected error defined in the testcase.
	if (!p.isErrorOnAfter() && !p.isErrorOnBefore()) ||
		tc.shouldPrintUsage ||
		(!p.isErrorOnBefore() && p.isErrorOnAfter() && tc.expectedErr == errExpectedFromCommand) {
		compareErrors(t, err, tc.expectedErr)
	}

	// IF
	// we EXPECT an error on Before
	// OR
	// we EXPECT an error on After AND the command was NOT EXPECTED to error
	// AND
	// we DON'T EXPECT the usage to be printed (<nil> or empty)
	// THEN
	// check we got the expected error from Before/After.
	if (p.isErrorOnBefore() ||
		(p.isErrorOnAfter() && tc.expectedErr != errExpectedFromCommand)) &&
		!tc.shouldPrintUsage {
		compareErrors(t, err, errExpected)
	}

	if printUsage != tc.shouldPrintUsage {
		t.Fatalf("expected printUsage to be %t got %t", tc.shouldPrintUsage, printUsage)
	}
}
