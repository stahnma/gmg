package emulator

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

// Serve answers grill commands on conn until ctx is cancelled, and runs the
// simulation clock at one tick per second.
func Serve(ctx context.Context, conn net.PacketConn, g *Grill, log *slog.Logger) error {
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				g.Tick()
			}
		}
	}()
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		req := string(buf[:n])
		cmd := protocol.ParseCommand(req)
		if cmd.Kind == protocol.KindUnknown {
			log.Warn("unsupported request", "from", addr, "request", req)
			continue
		}
		resp := g.Handle(cmd)
		if resp == nil {
			continue
		}
		if _, err := conn.WriteTo(resp, addr); err != nil {
			log.Error("send failed", "to", addr, "err", err)
			continue
		}
		log.Info("request", "from", addr, "request", req, "response_bytes", len(resp))
	}
}
