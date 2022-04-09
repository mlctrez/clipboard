package api

import "time"

// StorageApi defines the storage operations
type StorageApi interface {
	List() (timestamps []string, err error)
	Save(clip *ClippedImage) (err error)
	Get(timestamp string) (clip *ClippedImage, err error)
	Delete(timestamp string) (err error)
	Open(path string, timeout time.Duration) (err error)
	Close() (err error)
}
