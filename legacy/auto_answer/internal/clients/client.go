package clients

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"auto_answer/internal/domain"
)

type Client struct {
	serverURL *url.URL
	oz        domain.AuthenticationOzonApi
	wb        domain.AuthenticationWildberriesApi
}

type serverURLKey struct{}

type DecodeBodyError struct {
	Status      string
	ContentType string
	Body        []byte
	Err         error
}

func (d *DecodeBodyError) Error() string {
	return fmt.Sprintf("декодирование %s: %s", d.ContentType, d.Err)
}

func trimTrailingSlashes(u *url.URL) {
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
}

func NewOzClient(serverURL string, auth domain.AuthenticationOzonApi) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	trimTrailingSlashes(u)

	return &Client{
		serverURL: u,
		oz:        auth,
	}, nil
}

func NewWbClient(serverURL string, auth domain.AuthenticationWildberriesApi) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	trimTrailingSlashes(u)

	return &Client{
		serverURL: u,
		wb:        auth,
	}, nil
}

func (c *Client) requestURL(ctx context.Context) *url.URL {
	u, ok := ctx.Value(serverURLKey{}).(*url.URL)
	if !ok {
		return c.serverURL
	}
	return u
}
