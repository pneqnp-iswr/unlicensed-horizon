package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const maxStatusBytes = 1 << 20

type ServerProbeResult struct {
	Address         string          `json:"address"`
	TCPReachable    bool            `json:"tcp_reachable"`
	StatusResponded bool            `json:"status_responded"`
	Version         string          `json:"version,omitempty"`
	Protocol        int             `json:"protocol,omitempty"`
	PlayersOnline   int             `json:"players_online"`
	PlayersMax      int             `json:"players_max"`
	Description     json.RawMessage `json:"description,omitempty"`
	OnlineMode      string          `json:"online_mode"`
	Note            string          `json:"note"`
}

func serverAddress(raw string) (string, string, uint16, error) {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, "/\\@?# \t\n\r") {
		return "", "", 0, errors.New("expected HOST[:PORT], without URLs, spaces, or credentials")
	}
	host, portStr := raw, "25565"
	if strings.HasPrefix(raw, "[") {
		if strings.HasSuffix(raw, "]") {
			host = raw[1 : len(raw)-1]
		} else {
			h, p, e := net.SplitHostPort(raw)
			if e != nil {
				return "", "", 0, fmt.Errorf("invalid bracketed IPv6: %w", e)
			}
			host, portStr = h, p
		}
		if ip := net.ParseIP(host); ip == nil || !strings.Contains(host, ":") {
			return "", "", 0, errors.New("brackets must contain an IPv6 address")
		}
	} else {
		if strings.Count(raw, ":") > 1 {
			return "", "", 0, errors.New("IPv6 requires brackets, e.g. [::1]:25565")
		}
		if h, p, ok := strings.Cut(raw, ":"); ok {
			host, portStr = h, p
		}
		if host == "" || len(host) > 253 {
			return "", "", 0, errors.New("invalid hostname")
		}
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p < 1 || p > 65535 {
		return "", "", 0, errors.New("port must be 1..65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(p)), host, uint16(p), nil
}

func writeVarInt(dst *bytes.Buffer, n int32) {
	v := uint32(n)
	for v >= 0x80 {
		dst.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	dst.WriteByte(byte(v))
}

func readVarInt(r io.Reader) (int32, error) {
	var value uint32
	var b [1]byte
	for i := 0; i < 5; i++ {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= uint32(b[0]&0x7f) << (7 * i)
		if b[0]&0x80 == 0 {
			if i == 4 && b[0] > 0x0f {
				return 0, errors.New("invalid varint")
			}
			return int32(value), nil
		}
	}
	return 0, errors.New("varint too long")
}

// ProbeMinecraftStatus queries the unauthenticated Minecraft server-list endpoint.
// The status response is NOT evidence of a successful player login.
func ProbeMinecraftStatus(raw string, timeout time.Duration) (ServerProbeResult, error) {
	result := ServerProbeResult{
		OnlineMode: "unknown",
		Note:       "Status ping is not a login. Joining may require Microsoft authentication, a whitelist and matching game version.",
	}
	address, host, port, err := serverAddress(raw)
	if err != nil {
		return result, err
	}
	result.Address = address
	if timeout <= 0 || timeout > 30*time.Second {
		return result, errors.New("timeout must be 1ms..30s")
	}
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return result, fmt.Errorf("TCP connection to %s failed: %w", address, err)
	}
	defer conn.Close()
	result.TCPReachable = true
	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return result, err
	}

	var body bytes.Buffer
	writeVarInt(&body, 0) // handshake packet
	writeVarInt(&body, -1) // status is version-independent
	writeVarInt(&body, int32(len(host)))
	body.WriteString(host)
	body.WriteByte(byte(port >> 8))
	body.WriteByte(byte(port))
	writeVarInt(&body, 1) // next state: status
	var wire bytes.Buffer
	writeVarInt(&wire, int32(body.Len()))
	wire.Write(body.Bytes())
	wire.Write([]byte{1, 0}) // status request, packet ID 0
	if _, err = conn.Write(wire.Bytes()); err != nil {
		return result, fmt.Errorf("status handshake: %w", err)
	}
	length, err := readVarInt(conn)
	if err != nil {
		return result, fmt.Errorf("status frame: %w", err)
	}
	if length < 2 || length > maxStatusBytes {
		return result, fmt.Errorf("invalid status frame length %d", length)
	}
	frame := make([]byte, int(length))
	if _, err = io.ReadFull(conn, frame); err != nil {
		return result, fmt.Errorf("status response: %w", err)
	}
	br := bytes.NewReader(frame)
	packetID, err := readVarInt(br)
	if err != nil || packetID != 0 {
		return result, fmt.Errorf("unexpected status packet id %d: %v", packetID, err)
	}
	jsonLen, err := readVarInt(br)
	if err != nil || jsonLen < 2 || int(jsonLen) > br.Len() {
		return result, errors.New("invalid status JSON length")
	}
	jsonData := make([]byte, int(jsonLen))
	if _, err = io.ReadFull(br, jsonData); err != nil {
		return result, err
	}
	var status struct {
		Version struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol"`
		} `json:"version"`
		Players struct {
			Online int `json:"online"`
			Max    int `json:"max"`
		} `json:"players"`
		Description json.RawMessage `json:"description"`
	}
	if err = json.Unmarshal(jsonData, &status); err != nil {
		return result, fmt.Errorf("invalid status JSON: %w", err)
	}
	result.StatusResponded = true
	result.Version = status.Version.Name
	result.Protocol = status.Version.Protocol
	result.PlayersOnline = status.Players.Online
	result.PlayersMax = status.Players.Max
	if json.Valid(status.Description) {
		result.Description = status.Description
	}
	return result, nil
}
