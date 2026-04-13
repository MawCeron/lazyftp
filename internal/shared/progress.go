package shared

import "io"

// ProgressReader envuelve un io.Reader reportando bytes leídos
type ProgressReader struct {
	Reader   io.Reader
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

// ProgressWriter envuelve un io.Writer reportando bytes escritos
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
