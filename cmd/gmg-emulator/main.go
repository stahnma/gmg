// Command gmg-emulator emulates a Green Mountain Grill on its UDP protocol.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/stahnma/gmg/internal/emulator"
)

func main() {
	var address string
	var port int
	flag.StringVar(&address, "address", "0.0.0.0", "address to listen on (* for all)")
	flag.StringVar(&address, "a", "0.0.0.0", "shorthand for --address")
	flag.IntVar(&port, "port", 8080, "UDP port to listen on")
	flag.IntVar(&port, "p", 8080, "shorthand for --port")
	flag.Parse()
	if address == "*" {
		address = "0.0.0.0"
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	conn, err := net.ListenPacket("udp4", net.JoinHostPort(address, fmt.Sprint(port)))
	if err != nil {
		log.Error("listen failed", "err", err)
		os.Exit(1)
	}
	log.Info("grill emulator listening", "addr", conn.LocalAddr())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := emulator.Serve(ctx, conn, emulator.New(), log); err != nil {
		log.Error("serve failed", "err", err)
		os.Exit(1)
	}
}
