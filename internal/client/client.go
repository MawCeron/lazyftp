package client

import "github.com/MawCeron/lazyftp/internal/model"

type Client interface {
	Connect(host, user, pass string, port int) error
	Disconnect() error
	List(path string) ([]model.FileInfo, error)
	Upload(localPath, remotePath string, progress func(int64)) error
	Download(remotePath, localPath string, progress func(int64)) error
	Mkdir(path string) error
	CurrentPath() string
	ChangePath(path string) error
}
