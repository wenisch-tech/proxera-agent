package tunnel

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wenisch-tech/proxera-agent/internal/config"
	"github.com/wenisch-tech/proxera-agent/internal/protocol"
	"github.com/wenisch-tech/proxera-agent/internal/proxy"
)

// wsSession tracks one proxied WebSocket connection to a local service.
type wsSession struct {
	conn   *websocket.Conn
	sendCh chan wsOutbound
}

// wsOutbound is a message queued for delivery to the local WebSocket service.
type wsOutbound struct {
	data   []byte
	binary bool
	close  bool
	code   int
	reason string
}

// wsHandshakeHeaders are managed by the WebSocket dialer itself and must not be
// forwarded to the local service when establishing the upstream connection.
var wsHandshakeHeaders = map[string]bool{
	"upgrade":                  true,
	"connection":               true,
	"sec-websocket-key":        true,
	"sec-websocket-version":    true,
	"sec-websocket-extensions": true,
}

type Client struct {
	cfg         config.Config
	log         *slog.Logger
	proxyClient *proxy.Client
	backoff     Backoff
	dialer      websocket.Dialer
	wsSessions  sync.Map // wsSessionID (string) → *wsSession
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
		errorsCh <- c.heartbeatLoop(ctx, conn)
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
			if err := conn.SetReadDeadline(time.Now().Add(c.cfg.HeartbeatTimeout)); err != nil {
				return err
			}
			c.log.Debug("received pong", slog.String("correlationId", frame.CorrelationID))
		case protocol.TypeRequest:
			request, err := protocol.DecodePayload[protocol.RequestPayload](frame)
			if err != nil {
				c.enqueueError(outbound, "REQUEST_PARSE_FAILED", err.Error(), frame.CorrelationID)
				continue
			}

			go func(f protocol.Frame, req protocol.RequestPayload) {
				sem <- struct{}{}
				defer func() { <-sem }()
				c.handleRequest(ctx, outbound, f.CorrelationID, req)
			}(frame, request)
		case protocol.TypeWsOpen:
			wsOpen, err := protocol.DecodePayload[protocol.WsOpenPayload](frame)
			if err != nil {
				c.log.Warn("failed to parse WS_OPEN payload", slog.Any("error", err))
				continue
			}
			go c.handleWsOpen(ctx, outbound, frame.CorrelationID, wsOpen)
		case protocol.TypeWsData:
			wsData, err := protocol.DecodePayload[protocol.WsDataPayload](frame)
			if err != nil {
				c.log.Warn("failed to parse WS_DATA payload", slog.Any("error", err))
				continue
			}
			data, err := base64.StdEncoding.DecodeString(wsData.Data)
			if err != nil {
				c.log.Warn("failed to decode WS_DATA payload", slog.String("wsSessionId", wsData.WsSessionID), slog.Any("error", err))
				continue
			}
			if v, ok := c.wsSessions.Load(wsData.WsSessionID); ok {
				select {
				case v.(*wsSession).sendCh <- wsOutbound{data: data, binary: wsData.Binary}:
				default:
					c.log.Warn("WS_DATA send channel full, dropping frame", slog.String("wsSessionId", wsData.WsSessionID))
				}
			}
		case protocol.TypeWsClose:
			wsClose, err := protocol.DecodePayload[protocol.WsClosePayload](frame)
			if err != nil {
				c.log.Warn("failed to parse WS_CLOSE payload", slog.Any("error", err))
				continue
			}
			if v, ok := c.wsSessions.LoadAndDelete(wsClose.WsSessionID); ok {
				select {
				case v.(*wsSession).sendCh <- wsOutbound{close: true, code: wsClose.Code, reason: wsClose.Reason}:
				default:
					c.log.Warn("WS_CLOSE send channel full, dropping close frame", slog.String("wsSessionId", wsClose.WsSessionID))
				}
			}
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

func (c *Client) heartbeatLoop(ctx context.Context, conn *websocket.Conn) error {
	// Send an initial PONG immediately so the server's stale-session timer starts
	// fresh rather than from the WebSocket handshake timestamp.
	if err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(c.cfg.HeartbeatTimeout)); err != nil {
		return fmt.Errorf("initial heartbeat pong failed: %w", err)
	}

	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(c.cfg.HeartbeatTimeout)); err != nil {
				return fmt.Errorf("heartbeat pong failed: %w", err)
			}
			c.log.Debug("sent heartbeat pong")
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

// handleWsOpen dials the local WebSocket service and sets up a bidirectional
// proxy between the local connection and the Proxera tunnel.
//
// Frame flow:
//   - Outbound (local → tunnel): goroutine reads from localConn and sends WS_DATA / WS_CLOSE frames.
//   - Inbound (tunnel → local): goroutine drains session.sendCh and writes to localConn.
func (c *Client) handleWsOpen(ctx context.Context, outbound chan<- protocol.Frame, wsSessionID string, payload protocol.WsOpenPayload) {
	targetURL := fmt.Sprintf("ws://%s:%d%s", payload.LocalHost, payload.LocalPort, payload.Path)
	if payload.QueryString != "" {
		targetURL += "?" + payload.QueryString
	}

	localConn, resp, err := websocket.DefaultDialer.DialContext(ctx, targetURL, c.buildWsHeaders(payload.Headers))
	if err != nil {
		code := 1011
		if resp != nil {
			code = resp.StatusCode
		}
		select {
		case outbound <- protocol.Frame{
			Type:          protocol.TypeWsOpenReject,
			CorrelationID: wsSessionID,
			Payload:       protocol.MustEncodePayload(map[string]any{"code": code, "reason": err.Error()}),
		}:
		case <-ctx.Done():
		}
		c.log.Warn("WS_OPEN: failed to dial local service",
			slog.String("wsSessionId", wsSessionID),
			slog.String("target", targetURL),
			slog.Any("error", err),
		)
		return
	}

	session := &wsSession{
		conn:   localConn,
		sendCh: make(chan wsOutbound, 64),
	}
	c.wsSessions.Store(wsSessionID, session)

	// Acknowledge the open to the server.
	select {
	case outbound <- protocol.Frame{
		Type:          protocol.TypeWsOpenAck,
		CorrelationID: wsSessionID,
		Payload:       protocol.MustEncodePayload(map[string]any{}),
	}:
	case <-ctx.Done():
		localConn.Close()
		c.wsSessions.Delete(wsSessionID)
		return
	}

	c.log.Debug("WS_OPEN_ACK sent", slog.String("wsSessionId", wsSessionID), slog.String("target", targetURL))

	// Close the local connection when the tunnel session context is cancelled,
	// which unblocks any goroutine blocked on localConn.ReadMessage().
	go func() {
		<-ctx.Done()
		localConn.Close()
	}()

	// Goroutine: local WS → tunnel (agent-to-server direction).
	go func() {
		defer c.wsSessions.Delete(wsSessionID)
		defer localConn.Close()
		for {
			msgType, data, err := localConn.ReadMessage()
			if err != nil {
				code, reason := websocket.CloseNormalClosure, ""
				if ce, ok := err.(*websocket.CloseError); ok {
					code, reason = ce.Code, ce.Text
				}
				select {
				case outbound <- protocol.Frame{
					Type:    protocol.TypeWsClose,
					Payload: protocol.MustEncodePayload(protocol.WsClosePayload{WsSessionID: wsSessionID, Code: code, Reason: reason}),
				}:
				case <-ctx.Done():
				}
				return
			}
			binary := msgType == websocket.BinaryMessage
			select {
			case outbound <- protocol.Frame{
				Type:    protocol.TypeWsData,
				Payload: protocol.MustEncodePayload(protocol.WsDataPayload{WsSessionID: wsSessionID, Data: base64.StdEncoding.EncodeToString(data), Binary: binary}),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Goroutine: tunnel → local WS (server-to-agent direction).
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-session.sendCh:
				if !ok {
					return
				}
				if out.close {
					_ = localConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(out.code, out.reason))
					localConn.Close()
					return
				}
				msgType := websocket.TextMessage
				if out.binary {
					msgType = websocket.BinaryMessage
				}
				if err := localConn.WriteMessage(msgType, out.data); err != nil {
					return
				}
			}
		}
	}()
}

// buildWsHeaders converts the header map from a WS_OPEN payload into an http.Header
// suitable for the gorilla/websocket dialer, skipping handshake-only headers that
// the dialer manages itself.
func (c *Client) buildWsHeaders(raw map[string]string) http.Header {
	h := http.Header{}
	for k, v := range raw {
		if wsHandshakeHeaders[strings.ToLower(k)] {
			continue
		}
		h.Set(k, v)
	}
	return h
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
