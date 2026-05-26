package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wenisch-tech/proxera-agent/internal/protocol"
)

var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"proxy-connection":    {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"trailers":            {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

type Client struct {
	httpClient *http.Client
}

func New(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Handle(ctx context.Context, req protocol.RequestPayload) (protocol.ResponsePayload, error) {
	start := time.Now()

	forwardPath := stripPrefix(req.Path, req.StripPrefix)
	targetURL := url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("%s:%d", req.LocalHost, req.LocalPort),
		Path:     forwardPath,
		RawQuery: req.QueryString,
	}

	decodedBody, err := decodeBody(req.Body)
	if err != nil {
		return protocol.ResponsePayload{}, fmt.Errorf("decode request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), bytes.NewReader(decodedBody))
	if err != nil {
		return protocol.ResponsePayload{}, fmt.Errorf("build local request: %w", err)
	}

	for k, vals := range req.Headers {
		if _, blocked := hopByHopHeaders[strings.ToLower(k)]; blocked {
			continue
		}
		for _, v := range vals {
			httpReq.Header.Add(k, v)
		}
	}

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return protocol.ResponsePayload{}, fmt.Errorf("local request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return protocol.ResponsePayload{}, fmt.Errorf("read local response body: %w", err)
	}

	responseHeaders := map[string][]string{}
	for k, values := range res.Header {
		if len(values) == 0 {
			continue
		}
		if _, blocked := hopByHopHeaders[strings.ToLower(k)]; blocked {
			continue
		}
		responseHeaders[k] = values
	}

	// Gzip-compress large unencoded bodies so the tunnel frame stays small.
	// The server forwards Content-Encoding to the browser, which decompresses.
	_, alreadyEncoded := responseHeaders["Content-Encoding"]
	if !alreadyEncoded && len(responseBody) > 32*1024 {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, werr := gz.Write(responseBody); werr == nil {
			if werr = gz.Close(); werr == nil {
				responseBody = buf.Bytes()
				responseHeaders["Content-Encoding"] = []string{"gzip"}
				delete(responseHeaders, "Content-Length")
			}
		}
	}

	return protocol.ResponsePayload{
		Status:    res.StatusCode,
		Headers:   responseHeaders,
		Body:      encodeBody(responseBody),
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

func stripPrefix(path, prefix string) string {
	if prefix == "" || path == "" {
		if path == "" {
			return "/"
		}
		return path
	}
	if strings.HasPrefix(path, prefix) {
		trimmed := strings.TrimPrefix(path, prefix)
		if trimmed == "" {
			return "/"
		}
		if !strings.HasPrefix(trimmed, "/") {
			return "/" + trimmed
		}
		return trimmed
	}
	return path
}

func decodeBody(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func encodeBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(raw)
}
