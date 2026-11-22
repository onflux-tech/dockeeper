//go:build linux

package collector

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func defaultGatewayProbe(ctx context.Context) (bool, uint32, string) {
	gateway := defaultGatewayIP()
	if gateway == "" {
		return false, 0, "none"
	}

	if latency, ok := icmpEcho(ctx, gateway); ok {
		return true, latency, "icmp"
	}

	for _, port := range []string{"53", "80"} {
		started := time.Now()
		conn, err := (&net.Dialer{Timeout: 1500 * time.Millisecond}).DialContext(ctx, "tcp", net.JoinHostPort(gateway, port))
		if err != nil {
			continue
		}
		_ = conn.Close()
		return true, uint32(time.Since(started).Milliseconds()), "tcp"
	}

	return false, 0, "none"
}

func defaultGatewayIP() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[1] != "00000000" {
			continue
		}
		value, err := strconv.ParseUint(fields[2], 16, 32)
		if err != nil || value == 0 {
			continue
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(value))
		return net.IP(buf).String()
	}
	return ""
}

func icmpEcho(ctx context.Context, gateway string) (uint32, bool) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("dockeeper")},
	}
	payload, err := message.Marshal(nil)
	if err != nil {
		return 0, false
	}

	deadline := time.Now().Add(2 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return 0, false
	}

	started := time.Now()
	if _, err := conn.WriteTo(payload, &net.IPAddr{IP: net.ParseIP(gateway)}); err != nil {
		return 0, false
	}

	buffer := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			return 0, false
		}
		reply, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), buffer[:n])
		if err != nil {
			continue
		}
		if reply.Type == ipv4.ICMPTypeEchoReply {
			return uint32(time.Since(started).Milliseconds()), true
		}
	}
}
