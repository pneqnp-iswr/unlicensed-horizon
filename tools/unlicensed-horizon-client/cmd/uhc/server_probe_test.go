package main

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestServerAddress(t *testing.T) {
	cases := []struct{ in, want string; ok bool }{
		{"localhost", "localhost:25565", true},
		{"127.0.0.1:12345", "127.0.0.1:12345", true},
		{"[::1]:25565", "[::1]:25565", true},
		{"[::1]", "[::1]:25565", true},
		{"http://bad:25565", "", false},
		{"bad:0", "", false},
		{"::1", "", false},
		{"example.com/hello", "", false},
		{"bad:99999", "", false},
	}
	for _, tc := range cases {
		a, _, _, err := serverAddress(tc.in)
		if (err == nil) != tc.ok || tc.ok && a != tc.want {
			t.Errorf("%q: addr=%q err=%v", tc.in, a, err)
		}
	}
}

func TestMinecraftStatusProbe(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		size, err := readVarInt(conn)
		if err != nil {
			done <- err
			return
		}
		payload := make([]byte, int(size))
		if _, err = io.ReadFull(conn, payload); err != nil {
			done <- err
			return
		}
		r := bytes.NewReader(payload)
		packet, _ := readVarInt(r)
		protocol, _ := readVarInt(r)
		if packet != 0 || protocol != -1 {
			done <- io.ErrUnexpectedEOF
			return
		}
		hslen, _ := readVarInt(r)
		host := make([]byte, int(hslen))
		_, _ = io.ReadFull(r, host)
		statePort := make([]byte, 2)
		_, _ = io.ReadFull(r, statePort)
		state, _ := readVarInt(r)
		if string(host) != "127.0.0.1" || state != 1 {
			done <- io.ErrUnexpectedEOF
			return
		}
		reqSize, _ := readVarInt(conn)
		reqID, _ := readVarInt(conn)
		if reqSize != 1 || reqID != 0 {
			done <- io.ErrUnexpectedEOF
			return
		}
		payloadJSON := `{"version":{"name":"26.3","protocol":777},"players":{"max":20,"online":2},"description":{"text":"Local test"}}`
		var response bytes.Buffer
		writeVarInt(&response, 0)
		writeVarInt(&response, int32(len(payloadJSON)))
		response.WriteString(payloadJSON)
		var frame bytes.Buffer
		writeVarInt(&frame, int32(response.Len()))
		frame.Write(response.Bytes())
		_, err = conn.Write(frame.Bytes())
		done <- err
	}()
	p, err := ProbeMinecraftStatus(listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !p.TCPReachable || !p.StatusResponded || p.Version != "26.3" || p.PlayersOnline != 2 || p.OnlineMode != "unknown" {
		t.Fatalf("unexpected probe: %+v", p)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

func TestMalformedStatus(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e == nil {
			_, _ = c.Write([]byte{0})
			_ = c.Close()
		}
	}()
	r, err := ProbeMinecraftStatus(ln.Addr().String(), time.Second)
	if err == nil || !r.TCPReachable || r.StatusResponded || !strings.Contains(err.Error(), "status frame") {
		t.Fatalf("unexpected outcome: %+v %v", r, err)
	}
}
