package tunnel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wenisch-tech/proxera-client/internal/config"
	"github.com/wenisch-tech/proxera-client/internal/protocol"
	"github.com/wenisch-tech/proxera-client/internal/proxy"
)

type Client struct {
	cfg         config.Config
	log         *slog.Logger
	proxyClient *proxy.Client
	backoff     Backoff
	dialer      websocket.Dialer
}

func NewClient(cfg config.Config, logger *slog.Logger) *Client {
	return &Client{
		cfg:         cfg,
		log:         logger,
		proxyClient: proxy.New(cfg.RequestTimeout),
		backoff:     NewBackoff(cfg.ReconnectBase, cfg.ReconnectMax, time.Now().UnixNano()),
		dialer:      websocket.Dialer{},
	}
}

func (c *Client) Run(ctx context.Context) error {
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := c.runSession(ctx)
		if err == nil {
			attempt = 0
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		attempt++
		sleepFor := c.backoff.Duration(attempt)
		c.log.Warn("tunnel session ended; reconnect scheduled",
			slog.Any("error", err),
			slog.Int("attempt", attempt),
			slog.Duration("sleep", sleepFor),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepFor):
		}
	}
}

func (c *Client) runSession(ctx context.Context) error {
	headers := http.Header{}
	headers.Set("X-Proxera-Token", c.cfg.APIKey)

	conn, _, err := c.dialer.DialContext(ctx, c.cfg.ServerURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(c.cfg.HeartbeatTimeout)); err != nil {
		return err
	}
	conn.SetPongHandler(func(_ string) error {
		return conn.SetReadDeadline(time.Now().Add(c.cfg.HeartbeatTimeout))
	})

	c.log.Info("websocket connected", slog.String("serverURL", sanitizeURL(c.cfg.ServerURL)))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	outbound := make(chan protocol.Frame, c.cfg.ConcurrencyLimit*2)
	errorsCh := make(chan error, 2)
	sem := make(chan struct{}, c.cfg.ConcurrencyLimit)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		errorsCh <- c.readLoop(ctx, conn, outbound, sem, &wg)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		errorsCh <- c.writeLoop(ctx, conn, outbound)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		errorsCh <- c.heartbeatLoop(ctx, outbound)
	}()

	select {
	case <-ctx.Done():
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"), time.Now().Add(2*time.Second))
		wg.Wait()
		return ctx.Err()
	case err := <-errorsCh:
		cancel()
		wg.Wait()
		return err
	}
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, outbound chan<- protocol.Frame, sem chan struct{}, wg *sync.WaitGroup) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		frame, err := protocol.UnmarshalFrame(payload)
		if err != nil {
			c.enqueueError(outbound, "BAD_FRAME", err.Error(), "")
			continue
		}

		switch frame.Type {
		case protocol.TypeRegisterAck:
			ack, err := protocol.DecodePayload[protocol.RegisterAckPayload](frame)
			if err != nil {
				c.enqueueError(outbound, "REGISTER_ACK_PARSE_FAILED", err.Error(), frame.CorrelationID)
				continue
			}
			c.log.Info("registered to proxera", slog.String("clientId", ack.ClientID), slog.String("name", ack.Name))
		case protocol.TypePing:
			outbound <- protocol.Frame{Type: protocol.TypePong, CorrelationID: frame.CorrelationID}
		case protocol.TypePong:
			c.log.Debug("received pong", slog.String("correlationId", frame.CorrelationID))
		case protocol.TypeRequest:
			request, err := protocol.DecodePayload[protocol.RequestPayload](frame)
			if err != nil {
				c.enqueueError(outbound, "REQUEST_PARSE_FAILED", err.Error(), frame.CorrelationID)
				continue
			}

			sem <- struct{}{}
			go func(f protocol.Frame, req protocol.RequestPayload) {
				defer func() { <-sem }()
				c.handleRequest(ctx, outbound, f.CorrelationID, req)
			}(frame, request)
		case protocol.TypeError:
			errPayload, _ := protocol.DecodePayload[protocol.ErrorPayload](frame)
			c.log.Warn("received error frame from server",
				slog.String("code", errPayload.Code),
				slog.String("message", errPayload.Message),
				slog.String("correlationId", frame.CorrelationID),
			)
		default:
			c.enqueueError(outbound, "UNKNOWN_FRAME_TYPE", "unsupported frame type", frame.CorrelationID)
		}
	}
}

func (c *Client) writeLoop(ctx context.Context, conn *websocket.Conn, outbound <-chan protocol.Frame) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case frame := <-outbound:
			bytes, err := protocol.MarshalFrame(frame)
			if err != nil {
				c.log.Error("marshal outbound frame failed", slog.Any("error", err), slog.String("type", frame.Type))
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
				return err
			}
		}
	}
}

func (c *Client) heartbeatLoop(ctx context.Context, outbound chan<- protocol.Frame) error {
	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			outbound <- protocol.Frame{Type: protocol.TypePing, CorrelationID: fmt.Sprintf("%d", time.Now().UTC().UnixNano())}
			c.log.Debug("sent ping")
		}
	}
}

func (c *Client) handleRequest(ctx context.Context, outbound chan<- protocol.Frame, correlationID string, req protocol.RequestPayload) {
	response, err := c.proxyClient.Handle(ctx, req)
	if err != nil {
		c.enqueueError(outbound, "PROXY_FAILED", err.Error(), correlationID)
		c.log.Error("request proxy failed",
			slog.String("correlationId", correlationID),
			slog.String("method", req.Method),
			slog.String("path", req.Path),
			slog.Any("error", err),
		)
		return
	}

	outbound <- protocol.Frame{
		Type:          protocol.TypeResponse,
		CorrelationID: correlationID,
		Payload:       protocol.MustEncodePayload(response),
	}

	c.log.Info("proxied request",
		slog.String("correlationId", correlationID),
		slog.String("method", req.Method),
		slog.String("path", req.Path),
		slog.String("target", req.LocalHost),
		slog.Int("targetPort", req.LocalPort),
		slog.Int("status", response.Status),
		slog.Int64("latencyMs", response.LatencyMs),
	)
}

func (c *Client) enqueueError(outbound chan<- protocol.Frame, code, message, correlationID string) {
	outbound <- protocol.Frame{
		Type:          protocol.TypeError,
		CorrelationID: correlationID,
		Payload: protocol.MustEncodePayload(protocol.ErrorPayload{
			Code:    code,
			Message: message,
		}),
	}
}

func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.User != nil {
		parsed.User = url.User("***")
	}
	if parsed.RawQuery != "" {
		parsed.RawQuery = redactSensitiveQuery(parsed.Query().Encode())
	}
	return parsed.String()
}

func redactSensitiveQuery(raw string) string {
	parts := strings.Split(raw, "&")
	for i := range parts {
		kv := strings.SplitN(parts[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		if strings.Contains(key, "token") || strings.Contains(key, "key") || strings.Contains(key, "secret") {
			parts[i] = kv[0] + "=***"
		}
	}
	return strings.Join(parts, "&")
}
