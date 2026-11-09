package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/pion/dtls/v3"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/udp/coder"
)

// A sequential, bounded CoAP exchange over authenticated DTLS. We use the
// library's wire codec, avoiding its UDP client's unaligned ARM atomics.
// The caller serializes exchanges. Only fixed IKEA resource paths are used.
type ikeaTransport struct {
	conn ikeaConnection
	mid  uint16
}

type ikeaConnection interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

func ikeaDial(address, identity, key string) (*ikeaTransport, error) {
	ip := net.ParseIP(address)
	if ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
		return nil, errors.New("use the gateway's private IP address")
	}
	c, err := dtls.DialWithOptions("udp", &net.UDPAddr{IP: ip, Port: 5684},
		dtls.WithPSK(func([]byte) ([]byte, error) { return []byte(key), nil }),
		dtls.WithPSKIdentityHint([]byte(identity)),
		dtls.WithCipherSuites(dtls.TLS_PSK_WITH_AES_128_CCM_8))
	if err != nil {
		return nil, errors.New("could not connect to IKEA gateway")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err = c.HandshakeContext(ctx); err != nil {
		c.Close()
		return nil, errors.New("IKEA gateway unavailable or credentials rejected")
	}
	var seed [2]byte
	if _, err = rand.Read(seed[:]); err != nil {
		c.Close()
		return nil, err
	}
	return &ikeaTransport{conn: c, mid: binary.BigEndian.Uint16(seed[:])}, nil
}

func (t *ikeaTransport) exchange(code codes.Code, path string, payload []byte, block *uint32) (message.Message, error) {
	t.mid++
	tok := make([]byte, 8)
	if _, err := rand.Read(tok); err != nil {
		return message.Message{}, err
	}
	m := message.Message{Code: code, Type: message.Confirmable, MessageID: int32(t.mid), Token: tok, Payload: payload}
	for _, part := range splitIKEAPath(path) {
		m.Options = append(m.Options, message.Option{ID: message.URIPath, Value: []byte(part)})
	}
	if len(payload) > 0 {
		m.Options = append(m.Options, message.Option{ID: message.ContentFormat, Value: []byte{byte(message.AppJSON)}})
	}
	if block != nil {
		var buf [4]byte
		m.Options, _, _ = m.Options.SetUint32(buf[:], message.Block2, *block)
	}
	packet := make([]byte, 2048)
	n, err := coder.DefaultCoder.Encode(m, packet)
	if err != nil {
		return message.Message{}, err
	}
	packet = packet[:n]
	deadline := time.Now().Add(10 * time.Second)
	acknowledged := false
	for attempt := 0; attempt < 3; attempt++ {
		if !acknowledged {
			t.conn.SetWriteDeadline(deadline)
			if _, err = t.conn.Write(packet); err != nil {
				break
			}
		}
		until := time.Now().Add(time.Duration(2<<attempt) * time.Second)
		if until.After(deadline) {
			until = deadline
		}
		t.conn.SetReadDeadline(until)
		for time.Now().Before(until) {
			buf := make([]byte, 4096)
			n, e := t.conn.Read(buf)
			if e != nil {
				err = e
				break
			}
			r := message.Message{Options: make(message.Options, 0, 32)}
			if _, e = coder.DefaultCoder.Decode(buf[:n], &r); e != nil {
				continue
			}
			if r.Type == message.Reset && r.MessageID == m.MessageID {
				return r, errors.New("IKEA gateway rejected request")
			}
			if r.Type == message.Acknowledgement && r.MessageID == m.MessageID && r.Code == codes.Empty {
				acknowledged = true
				continue
			}
			if !bytes.Equal(r.Token, tok) || r.Code < 64 {
				continue
			}
			if r.Type == message.Acknowledgement && r.MessageID != m.MessageID {
				continue
			}
			if r.Type == message.Confirmable {
				ack := message.Message{Type: message.Acknowledgement, MessageID: r.MessageID, Code: codes.Empty}
				b := make([]byte, 4)
				size, _ := coder.DefaultCoder.Encode(ack, b)
				t.conn.Write(b[:size])
			}
			if r.Code < 64 || r.Code >= 96 {
				return r, fmtIKEAResponse(r.Code)
			}
			return r, nil
		}
	}
	return message.Message{}, errors.New("IKEA gateway did not respond")
}

func fmtIKEAResponse(code codes.Code) error {
	return errors.New("IKEA gateway rejected request (" + strconv.Itoa(int(code)) + ")")
}

func splitIKEAPath(path string) []string {
	// Paths originate only from fixed resource numbers and validated numeric IDs.
	parts := []string{}
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	return parts
}

func (t *ikeaTransport) request(code codes.Code, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	var result []byte
	var etag []byte
	var block *uint32
	for count := 0; count < 128; count++ {
		r, err := t.exchange(code, path, payload, block)
		if err != nil {
			return err
		}
		b, e := r.Options.GetUint32(message.Block2)
		if e == nil {
			size := 1 << ((b & 7) + 4)
			if b&7 > 6 || int(b>>4)*size != len(result) {
				return errors.New("invalid IKEA block response")
			}
			for _, o := range r.Options {
				if o.ID == message.ETag {
					if count > 0 && !bytes.Equal(etag, o.Value) {
						return errors.New("IKEA response changed during transfer")
					}
					etag = append([]byte{}, o.Value...)
				}
			}
		}
		result = append(result, r.Payload...)
		if len(result) > 128*1024 {
			return errors.New("IKEA response too large")
		}
		if e != nil || b&8 == 0 {
			if out != nil {
				return json.Unmarshal(result, out)
			}
			return nil
		}
		if code != codes.GET {
			return errors.New("unexpected IKEA block response")
		}
		next := ((b>>4)+1)<<4 | (b & 7)
		block = &next
	}
	return errors.New("too many IKEA response blocks")
}
