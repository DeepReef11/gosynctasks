package todoist

import (
	"bytes"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// mockCommand creates a cobra.Command with specified flags for testing
type mockCommand struct {
	*cobra.Command
	flags map[string]interface{}
}

// newMockCommand creates a mock cobra command with common flags used in operations
func newMockCommand() *mockCommand {
	cmd := &cobra.Command{
		Use: "test",
	}

	// Add all flags that operations might use
	cmd.Flags().String("description", "", "Task description")
	cmd.Flags().Int("priority", 0, "Task priority")
	cmd.Flags().String("add-status", "", "Status for add action")
	cmd.Flags().StringArray("status", []string{}, "Status for update/complete")
	cmd.Flags().String("due-date", "", "Due date")
	cmd.Flags().String("start-date", "", "Start date")
	cmd.Flags().String("parent", "", "Parent task")
	cmd.Flags().Bool("literal", false, "Literal mode")
	cmd.Flags().String("summary", "", "Task summary")
	cmd.Flags().String("view", "", "View name")

	return &mockCommand{
		Command: cmd,
		flags:   make(map[string]interface{}),
	}
}

// withFlag sets a flag value on the mock command
func (m *mockCommand) withFlag(name string, value interface{}) *mockCommand {
	switch v := value.(type) {
	case string:
		_ = m.Command.Flags().Set(name, v)
	case int:
		_ = m.Command.Flags().Set(name, string(rune(v)))
	case bool:
		if v {
			_ = m.Command.Flags().Set(name, "true")
		}
	case []string:
		for _, s := range v {
			_ = m.Command.Flags().Set(name, s)
		}
	}
	return m
}

// mockSyncProvider is a test sync provider that doesn't spawn background processes
// It works by returning nil from GetSyncCoordinator(), which prevents triggerPushSync
// from spawning background sync processes during tests
type mockSyncProvider struct {
	syncCoordinator interface{}
}

func newMockSyncProvider() *mockSyncProvider {
	return &mockSyncProvider{}
}

func (m *mockSyncProvider) GetSyncCoordinator() interface{} {
	return m.syncCoordinator // Always nil - prevents background sync spawning
}

// captureOutput captures stdout/stderr during test execution
type outputCapture struct {
	oldStdout *os.File
	oldStderr *os.File
	r         *os.File
	w         *os.File
	outC      chan string
}

// newOutputCapture creates a new output capture
func newOutputCapture() *outputCapture {
	return &outputCapture{}
}

// start begins capturing output
func (o *outputCapture) start() {
	o.oldStdout = os.Stdout
	o.oldStderr = os.Stderr
	o.r, o.w, _ = os.Pipe()
	os.Stdout = o.w
	os.Stderr = o.w

	o.outC = make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, o.r)
		o.outC <- buf.String()
	}()
}

// stop stops capturing and returns the captured output
func (o *outputCapture) stop() string {
	_ = o.w.Close()
	os.Stdout = o.oldStdout
	os.Stderr = o.oldStderr
	out := <-o.outC
	return out
}
