package ui

import (
	"strings"
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

func TestACancelledTransferIsNotShownAsAnError(t *testing.T) {
	p := NewProcessesPanel()
	p, _ = p.Update(shared.TransferStartMsg{Transfer: shared.Transfer{ID: 1, Filename: "big.iso", Total: 100}})

	p, _ = p.Update(shared.TransferCancelledMsg{ID: 1, Filename: "big.iso"})

	if got := p.transfers[0].Status; got != shared.StatusCancelled {
		t.Fatalf("status = %v, want StatusCancelled", got)
	}
	row := renderTransfer(p.transfers[0], 60)
	if !strings.Contains(row, "cancelled") {
		t.Errorf("row %q does not say it was cancelled", row)
	}
	if strings.Contains(row, iconError()) {
		t.Errorf("row %q carries the error mark", row)
	}
}

func TestCancellingOneRowLeavesItsNamesakeAlone(t *testing.T) {
	p := NewProcessesPanel()
	for _, id := range []int64{1, 2} {
		p, _ = p.Update(shared.TransferStartMsg{Transfer: shared.Transfer{ID: id, Filename: "index.html", Total: 100}})
	}

	p, _ = p.Update(shared.TransferCancelledMsg{ID: 1, Filename: "index.html"})

	if p.transfers[0].Status != shared.StatusCancelled || p.transfers[1].Status != shared.StatusInProgress {
		t.Errorf("statuses = %v, %v; want cancelled, in progress", p.transfers[0].Status, p.transfers[1].Status)
	}
}

func TestACancelledTransferRefreshesBothPanels(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.processes = a.processes.AddTransfer(shared.Transfer{ID: 1, Filename: "big.iso", Total: 100})

	model, cmd := a.Update(shared.TransferCancelledMsg{ID: 1, Filename: "big.iso"})

	if got := model.(App).processes.transfers[0].Status; got != shared.StatusCancelled {
		t.Errorf("status = %v, want StatusCancelled", got)
	}
	if cmd == nil {
		t.Error("no listing was reloaded, so the removed partial file would stay on screen")
	}
}
