package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
)

// transferSeq hands out the ID that ties a transfer's Start/Progress/Error/Done
// messages together. Filename alone can't do this: re-transferring a file
// that shares a name with an earlier (possibly already-finished) row would
// update both by matching on Filename.
var transferSeq atomic.Int64

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
		if job.File.IsDir() && job.Direction == Upload {
			go m.guard(job, m.runDir)
		} else {
			go m.guard(job, m.run)
		}
	}
}

// These goroutines run outside Bubble Tea, whose own panic recovery never sees
// them. Recursive directory uploads share these frames, so one guard covers the
// whole tree.
func (m *Manager) guard(job Job, run func(Job)) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		p := m.program()
		if p == nil {
			return
		}

		// Otherwise the row sits at "in progress" for the rest of the session.
		p.Send(shared.TransferErrorMsg{
			Filename: job.File.Name,
			Err:      fmt.Errorf("%v", r),
		})
		p.Send(shared.LogMsg{
			Message: fmt.Sprintf("Transfer of %s failed: %v", job.File.Name, r),
			Level:   shared.LogError,
		})
	}()

	run(job)
}

func (m *Manager) runDir(job Job) {
	p := m.program()
	if p == nil {
		return
	}

	localDirPath := filepath.Join(job.LocalPath, job.File.Name)
	remoteDirPath := filepath.Join(job.RemotePath, job.File.Name)

	if err := m.client.Mkdir(remoteDirPath); err != nil {
		p.Send(shared.LogMsg{
			Message: fmt.Sprintf("Error creating remote directory %s: %v", job.File.Name, err),
			Level:   shared.LogError,
		})
		return
	}

	p.Send(shared.LogMsg{
		Message: fmt.Sprintf("Directory created: %s", remoteDirPath),
		Level:   shared.LogInfo,
	})

	entries, err := os.ReadDir(localDirPath)
	if err != nil {
		p.Send(shared.LogMsg{
			Message: fmt.Sprintf("Error reading directory %s: %v", localDirPath, err),
			Level:   shared.LogError,
		})
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileType := model.FileTypeFile
		if entry.IsDir() {
			fileType = model.FileTypeDir
		}

		subJob := Job{
			File: model.FileInfo{
				Name: entry.Name(),
				Size: info.Size(),
				Type: fileType,
			},
			LocalPath:  localDirPath,
			RemotePath: remoteDirPath,
			Direction:  Upload,
		}

		if entry.IsDir() {
			m.runDir(subJob)
		} else {
			m.run(subJob)
		}
	}
}

func (m *Manager) run(job Job) {
	p := m.program()
	if p == nil {
		return
	}

	filename := job.File.Name
	id := transferSeq.Add(1)

	direction := shared.DirectionUpload
	if job.Direction == Download {
		direction = shared.DirectionDownload
	}

	p.Send(shared.TransferStartMsg{
		Transfer: shared.Transfer{
			ID:        id,
			Filename:  filename,
			Total:     job.File.Size,
			Direction: direction,
			Status:    shared.StatusInProgress,
		},
	})

	progress := func(current int64) {
		p.Send(shared.TransferProgressMsg{
			ID:       id,
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
			ID:       id,
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
		p.Send(shared.TransferDoneMsg{ID: id, Filename: filename})
	}
}
