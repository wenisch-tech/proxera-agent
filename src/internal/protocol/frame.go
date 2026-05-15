package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	TypeRegisterAck = "REGISTER_ACK"
	TypeRequest     = "REQUEST"
	TypeResponse    = "RESPONSE"
	TypePing        = "PING"
	TypePong        = "PONG"
	TypeError       = "ERROR"
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
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	QueryString   string            `json:"queryString,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	LocalHost     string            `json:"localHost"`
	LocalPort     int               `json:"localPort"`
	StripPrefix   string            `json:"stripPrefix,omitempty"`
	RemoteAddress string            `json:"remoteAddress,omitempty"`
}

type ResponsePayload struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	LatencyMs int64             `json:"latencyMs"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
