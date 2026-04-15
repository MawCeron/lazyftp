package shared

// TransferDirection indicates the direction of a transfer
type TransferDirection int

const (
	DirectionUpload TransferDirection = iota
	DirectionDownload
)

// TransferStatus indicates the status of a transfer
type TransferStatus int

const (
	StatusInProgress TransferStatus = iota
	StatusDone
	StatusError
)

// Transfer represents an active transfer
type Transfer struct {
	Filename  string
	Total     int64
	Current   int64
	Direction TransferDirection
	Status    TransferStatus
}

func (t Transfer) Progress() float64 {
	if t.Total == 0 {
		return 0
	}
	return float64(t.Current) / float64(t.Total)
}

type TransferStartMsg struct {
	Transfer Transfer
}

type TransferProgressMsg struct {
	Filename string
	Current  int64
}

type TransferErrorMsg struct {
	Filename string
	Err      error
}

type LogLevel int

const (
	LogInfo LogLevel = iota
	LogSuccess
	LogError
)

type LogMsg struct {
	Message string
	Level   LogLevel
}

// TransferDoneMsg emits when a transfer ends successfully
type TransferDoneMsg struct {
	Filename string
}
