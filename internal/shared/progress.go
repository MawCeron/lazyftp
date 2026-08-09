package shared

import "io"

// ProgressReader wraps an io.ReadSeeker reporting bytes read.
//
// It has to be a seeker, not merely a reader: goftp resumes an interrupted
// upload only when the source can seek back to where the server got to, and it
// checks for that by type assertion. Wrapping a file that can seek in something
// that cannot is enough to turn resuming off.
type ProgressReader struct {
	Reader   io.ReadSeeker
	Total    int64
	Current  int64
	Callback func(int64)
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.Current += int64(n)
	if r.Callback != nil {
		r.Callback(r.Current)
	}
	return n, err
}

func (r *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := r.Reader.Seek(offset, whence)
	if err != nil {
		return pos, err
	}

	// Progress follows the file rather than the bytes handed over. A resume
	// starts again from what the server already holds, which is usually less
	// than what was read before the transfer broke; counting on from the old
	// total would report more sent than ever arrived.
	r.Current = pos
	if r.Callback != nil {
		r.Callback(r.Current)
	}

	return pos, nil
}

// ProgressWriter wraps an io.Reader reporting bytes written
type ProgressWriter struct {
	Writer   io.Writer
	Total    int64
	Current  int64
	Callback func(int64)
}

func (w *ProgressWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.Current += int64(n)
	if w.Callback != nil {
		w.Callback(w.Current)
	}
	return n, err
}
