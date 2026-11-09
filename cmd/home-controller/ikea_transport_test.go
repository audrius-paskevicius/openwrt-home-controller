package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/udp/coder"
)

type fakeIKEAConnection struct {
	handle  func(message.Message) []message.Message
	replies [][]byte
	writes  []message.Message
}

func (c *fakeIKEAConnection) Close() error                     { return nil }
func (c *fakeIKEAConnection) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeIKEAConnection) SetWriteDeadline(time.Time) error { return nil }
func (c *fakeIKEAConnection) Read(b []byte) (int, error) {
	if len(c.replies) == 0 {
		return 0, errors.New("simulated timeout")
	}
	r := c.replies[0]
	c.replies = c.replies[1:]
	return copy(b, r), nil
}
func (c *fakeIKEAConnection) Write(b []byte) (int, error) {
	m := message.Message{Options: make(message.Options, 0, 32)}
	if _, err := coder.DefaultCoder.Decode(b, &m); err != nil {
		return 0, err
	}
	c.writes = append(c.writes, m)
	for _, r := range c.handle(m) {
		buf := make([]byte, 4096)
		n, err := coder.DefaultCoder.Encode(r, buf)
		if err != nil {
			return 0, err
		}
		c.replies = append(c.replies, buf[:n])
	}
	return len(b), nil
}
func TestIKEAExchangeRetransmissionAndSeparateResponse(t *testing.T) {
	conn := &fakeIKEAConnection{}
	attempts := 0
	conn.handle = func(m message.Message) []message.Message {
		if m.Type == message.Acknowledgement {
			return nil
		}
		attempts++
		if attempts == 1 {
			return nil
		}
		return []message.Message{
			{Type: message.Acknowledgement, Code: codes.Empty, MessageID: m.MessageID},
			{Type: message.Confirmable, Code: codes.Content, MessageID: 100, Token: m.Token, Payload: []byte(`[1,2]`), Options: message.Options{{ID: message.ContentFormat, Value: []byte{50}}}},
		}
	}
	transport := &ikeaTransport{conn: conn}
	var got []int
	if err := transport.request(codes.GET, "/15001", nil, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(conn.writes) != 3 || conn.writes[0].MessageID != conn.writes[1].MessageID || conn.writes[2].Type != message.Acknowledgement {
		t.Fatal("retransmission/separate response handling", got, conn.writes)
	}
}
func TestIKEABlockwiseAndCommandSerialization(t *testing.T) {
	payload := `"` + strings.Repeat("x", 80) + `"`
	conn := &fakeIKEAConnection{}
	conn.handle = func(m message.Message) []message.Message {
		block, err := m.Options.GetUint32(message.Block2)
		start := 0
		if err == nil {
			start = int(block>>4) * 64
		}
		end := start + 64
		if end > len(payload) {
			end = len(payload)
		}
		option := uint32(start/64)<<4 | 2
		if end < len(payload) {
			option |= 8
		}
		var buf [4]byte
		opts, _, _ := message.Options{}.SetUint32(buf[:], message.Block2, option)
		return []message.Message{{Type: message.Acknowledgement, Code: codes.Content, Token: m.Token, MessageID: m.MessageID, Payload: []byte(payload[start:end]), Options: opts}}
	}
	transport := &ikeaTransport{conn: conn}
	var got string
	if err := transport.request(codes.GET, "/15001/1", nil, &got); err != nil {
		t.Fatal(err)
	}
	if got != strings.Repeat("x", 80) {
		t.Fatal("truncated blocks")
	}
	conn.handle = func(m message.Message) []message.Message {
		var v map[string]any
		if err := json.Unmarshal(m.Payload, &v); err != nil {
			t.Fatal(err)
		}
		path, _ := m.Options.Path()
		if path != "/15001/1" || m.Code != codes.PUT {
			t.Fatal(path, m.Code)
		}
		return []message.Message{{Type: message.Acknowledgement, Code: codes.Changed, Token: m.Token, MessageID: m.MessageID}}
	}
	if err := transport.request(codes.PUT, "/15001/1", map[string]int{"5850": 1}, nil); err != nil {
		t.Fatal(err)
	}
}
