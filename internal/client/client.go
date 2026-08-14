package client

import (
	"time"

	"github.com/MawCeron/lazyftp/internal/model"
)

// dialTimeout bounds how long establishing a connection may take. Left unset,
// a dial waits out the operating system's TCP timeout instead.
const dialTimeout = 10 * time.Second

type Client interface {
	Connect(host, user, pass string, port int) error
	Disconnect() error
	List(path string) ([]model.FileInfo, error)
	Upload(localPath, remotePath string, progress func(int64)) error
	Download(remotePath, localPath string, progress func(int64)) error
	Mkdir(path string) error
}
