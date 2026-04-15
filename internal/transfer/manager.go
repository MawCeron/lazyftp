package transfer

import (
	"fmt"
	"path/filepath"

	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	tea "github.com/charmbracelet/bubbletea"
)

type Direction int

const (
	Upload Direction = iota
	Download
)

type Job struct {
	File       model.FileInfo
	LocalPath  string
	RemotePath string
	Direction  Direction
}

type Manager struct {
	client  client.Client
	program func() *tea.Program
}

func NewManager(c client.Client, p func() *tea.Program) *Manager {
	return &Manager{
		client:  c,
		program: p,
	}
}

func (m *Manager) Enqueue(jobs []Job) {
	for _, job := range jobs {
		go m.run(job)
	}
}

func (m *Manager) run(job Job) {
	p := m.program()
	if p == nil {
		return
	}

	filename := job.File.Name

	direction := shared.DirectionUpload
	if job.Direction == Download {
		direction = shared.DirectionDownload
	}

	p.Send(shared.TransferStartMsg{
		Transfer: shared.Transfer{
			Filename:  filename,
			Total:     job.File.Size,
			Direction: direction,
			Status:    shared.StatusInProgress,
		},
	})

	progress := func(current int64) {
		p.Send(shared.TransferProgressMsg{
			Filename: filename,
			Current:  current,
		})
	}

	var err error
	switch job.Direction {
	case Upload:
		localFile := filepath.Join(job.LocalPath, filename)
		err = m.client.Upload(localFile, job.RemotePath, progress)
	case Download:
		remoteFile := filepath.Join(job.RemotePath, filename)
		err = m.client.Download(remoteFile, job.LocalPath, progress)
	}

	if err != nil {
		p.Send(shared.TransferErrorMsg{
			Filename: filename,
			Err:      err,
		})
		p.Send(shared.LogMsg{
			Message: fmt.Sprintf("Error: %s — %v", filename, err),
			Level:   shared.LogError,
		})
	} else {
		p.Send(shared.LogMsg{
			Message: fmt.Sprintf("Complete transfer: %s", filename),
			Level:   shared.LogSuccess,
		})
		p.Send(shared.TransferDoneMsg{Filename: filename})
	}
}
