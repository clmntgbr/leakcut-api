package port

import (
	"context"
	"io"
)

type RemoteObject struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}

type RemoteFetcher interface {
	Fetch(ctx context.Context, rawURL string) (*RemoteObject, error)
}
