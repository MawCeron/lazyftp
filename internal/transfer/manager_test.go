package transfer

import (
	"testing"

	"github.com/MawCeron/lazyftp/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

// Transfers run as bare goroutines, outside Bubble Tea's panic handling. An
// unrecovered panic in one of them ends the process with the terminal still in
// raw mode, so reaching the end of this test is the whole assertion.
func TestAPanickingTransferDoesNotTakeTheProcessDown(t *testing.T) {
	m := NewManager(nil, func() *tea.Program { return nil })

	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}}, func(Job) {
		panic("the server hung up mid-write")
	})
}

func TestGuardRunsTheTransfer(t *testing.T) {
	m := NewManager(nil, func() *tea.Program { return nil })

	var got string
	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}}, func(j Job) {
		got = j.File.Name
	})

	if got != "report.pdf" {
		t.Errorf("guard ran the transfer with %q, want \"report.pdf\"", got)
	}
}
