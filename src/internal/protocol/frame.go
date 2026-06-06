package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	TypeRegisterAck  = "REGISTER_ACK"
	TypeRequest      = "REQUEST"
	TypeResponse     = "RESPONSE"
	TypePing         = "PING"
	TypePong         = "PONG"
	TypeError        = "ERROR"
	TypeWsOpen       = "WS_OPEN"
	TypeWsOpenAck    = "WS_OPEN_ACK"
	TypeWsOpenReject = "WS_OPEN_REJECT"
	TypeWsData       = "WS_DATA"
	TypeWsClose      = "WS_CLOSE"
)

type Frame struct {
	Type          string          `json:"type"`
	CorrelationID string          `json:"correlationId,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

type RegisterAckPayload struct {
	ClientID string `json:"clientId"`
	Name     string `json:"name"`
}

type RequestPayload struct {
	Method        string              `json:"method"`
	Path          string              `json:"path"`
	QueryString   string              `json:"queryString,omitempty"`
	Headers       map[string][]string `json:"headers,omitempty"`
	Body          string              `json:"body,omitempty"`
	LocalHost     string              `json:"localHost"`
	LocalPort     int                 `json:"localPort"`
	StripPrefix   string              `json:"stripPrefix,omitempty"`
	RemoteAddress string              `json:"remoteAddress,omitempty"`
	PreserveHostHeader bool           `json:"preserveHostHeader,omitempty"`
}

type ResponsePayload struct {
	Status    int                 `json:"status"`
	Headers   map[string][]string `json:"headers,omitempty"`
	Body      string              `json:"body,omitempty"`
	LatencyMs int64               `json:"latencyMs"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WsOpenPayload is the payload of a WS_OPEN frame sent by the server to the agent.
type WsOpenPayload struct {
	WsSessionID string            `json:"wsSessionId"`
	LocalHost   string            `json:"localHost"`
	LocalPort   int               `json:"localPort"`
	Path        string            `json:"path"`
	QueryString string            `json:"queryString,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// WsDataPayload is the payload of a WS_DATA frame (bidirectional).
type WsDataPayload struct {
	WsSessionID string `json:"wsSessionId"`
	Data        string `json:"data"`
	Binary      bool   `json:"binary"`
}

// WsClosePayload is the payload of a WS_CLOSE frame (bidirectional).
type WsClosePayload struct {
	WsSessionID string `json:"wsSessionId"`
	Code        int    `json:"code"`
	Reason      string `json:"reason"`
}

func MarshalFrame(frame Frame) ([]byte, error) {
	if frame.Type == "" {
		return nil, fmt.Errorf("frame type is required")
	}
	return json.Marshal(frame)
}

func UnmarshalFrame(data []byte) (Frame, error) {
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return Frame{}, err
	}
	if frame.Type == "" {
		return Frame{}, fmt.Errorf("missing frame type")
	}
	return frame, nil
}

func DecodePayload[T any](frame Frame) (T, error) {
	var payload T
	if len(frame.Payload) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func MustEncodePayload(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	bytes, _ := json.Marshal(v)
	return bytes
}
