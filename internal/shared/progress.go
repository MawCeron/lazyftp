package shared

import "io"

// ProgressReader wraps an io.ReadSeeker reporting bytes read. It must be a
// seeker: goftp resumes an upload only when the source is one.
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

	// Progress follows the file, not the bytes read so far.
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
