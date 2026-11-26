// Command gmg is the Green Mountain Grill web controller: it polls the grill,
// records history, sends alerts and serves the web UI and HTTP API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stahnma/gmg/internal/alerts"
	"github.com/stahnma/gmg/internal/api"
	"github.com/stahnma/gmg/internal/config"
	"github.com/stahnma/gmg/internal/grill"
	"github.com/stahnma/gmg/internal/poller"
	"github.com/stahnma/gmg/internal/protocol"
	"github.com/stahnma/gmg/internal/store"
	"github.com/stahnma/gmg/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	envFile := flag.String("env-file", "", "read settings from this dotenv file (default: "+config.DefaultEnvFile+" in the working directory, if present)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [--env-file FILE]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Settings come from GMG_* environment variables, which override the same")
		fmt.Fprintln(flag.CommandLine.Output(), "names in the env file. See .env.example for the full list.")
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		return fmt.Errorf("unexpected arguments: %v", flag.Args())
	}

	path, required := *envFile, true
	if path == "" {
		path, required = config.DefaultEnvFile, false
	}
	getenv, loaded, err := config.WithEnvFile(os.LookupEnv, path, required)
	if err != nil {
		return err
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)
	if loaded {
		log.Info("settings loaded", "env_file", path)
	}

	db, err := store.Open(cfg.DBPath, cfg.DBMaxSizeMB)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Info("history database", "path", db.Path())

	client := grill.New(cfg.GrillHost, cfg.GrillPort, cfg.GrillTries, cfg.GrillRetry, log.With("component", "grill"))
	client.SetLimits(grill.Limits{
		Grill: grill.Range{Min: cfg.GrillTempMin, Max: cfg.GrillTempMax},
		Food:  grill.Range{Min: cfg.FoodTempMin, Max: cfg.FoodTempMax},
	})
	poll := poller.New(client, cfg.PollingInterval, log.With("component", "poller"))
	hub := api.NewHub()

	senders := []alerts.Sender{alerts.BrowserSender{Publish: hub.Publish, AudioEnabled: cfg.AudioEnabled}}
	if cfg.SlackWebhookURL != "" {
		senders = append(senders, alerts.SlackSender{WebhookURL: cfg.SlackWebhookURL, Channel: cfg.SlackChannel, Username: cfg.SlackUsername})
		log.Info("slack alerts enabled", "channel", cfg.SlackChannel)
	}
	handlers := append(alerts.DefaultHandlers(),
		&alerts.TempDrop{Degrees: cfg.TempDropDegrees, For: cfg.TempDropFor},
		&alerts.Ignition{Degrees: cfg.IgnitionDegrees, Within: cfg.IgnitionWithin},
	)
	alertManager := alerts.NewManager(handlers, senders, log.With("component", "alerts"))

	poll.Subscribe(func(s protocol.Status) { hub.Publish("status", s) })
	poll.Subscribe(func(s protocol.Status) { alertManager.OnStatus(s) })
	poll.Subscribe(func(s protocol.Status) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Record(ctx, s); err != nil { // never fatal
			log.Error("saving history failed", "err", err)
		}
	})

	unreachable := &alerts.Unreachable{After: cfg.UnreachableAfter}
	poll.OnHealth(func(h poller.Health, last protocol.Status, ok bool) {
		hub.Publish("connection", h)
		if a := unreachable.Check(h.Reachable, h.LastSeen, last, time.Now()); a != nil {
			alertManager.Send(*a)
		}
	})

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: (&api.Server{
			Grill: client, History: db, Hub: hub, LastStatus: poll.Last, Health: poll.Health,
			Static: web.Dist(), Log: log.With("component", "http"), AllowedOrigins: cfg.Origins(),
			Limits: client.Limits,
		}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go poll.Run(ctx)

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "grill", cfg.GrillHost, "audio", cfg.AudioEnabled)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	alertManager.Wait()
	return nil
}
