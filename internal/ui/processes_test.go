package ui

import (
	"testing"

	"github.com/MawCeron/lazyftp/internal/shared"
)

func TestSameNamedTransfersAreTrackedSeparately(t *testing.T) {
	p := NewProcessesPanel()
	for _, id := range []int64{1, 2} {
		p, _ = p.Update(shared.TransferStartMsg{Transfer: shared.Transfer{ID: id, Filename: "index.html", Total: 100}})
	}

	p, _ = p.Update(shared.TransferProgressMsg{ID: 1, Filename: "index.html", Current: 40})
	p, _ = p.Update(shared.TransferErrorMsg{ID: 2, Filename: "index.html"})

	first, second := p.transfers[0], p.transfers[1]
	if first.Current != 40 || first.Status != shared.StatusInProgress {
		t.Errorf("first row = %d bytes, status %v; want 40 bytes, in progress", first.Current, first.Status)
	}
	if second.Current != 0 || second.Status != shared.StatusError {
		t.Errorf("second row = %d bytes, status %v; want 0 bytes, error", second.Current, second.Status)
	}
}
