package main

import (
	"compliance-query-agent/internal/agent"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": agent.ErrorCode(err)})
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println(agent.Version)
		return nil
	}
	if len(args) < 1 || (args[0] != "query" && args[0] != "serve") {
		fmt.Fprintln(os.Stderr, "Usage: compliance-agent query --config configs/demo.json --input examples/mlps.json | serve --config configs/demo.json | version")
		return &agent.Error{Code: "USAGE", HTTPStatus: 400}
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "configs/demo.json", "configuration file")
	input := flags.String("input", "-", "JSON input file, or - for stdin; query only")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
		return &agent.Error{Code: "USAGE", HTTPStatus: 400}
	}
	cfg, err := agent.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	engine, err := agent.NewEngine(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if args[0] == "query" {
		var rd io.Reader = os.Stdin
		if *input != "-" {
			f, e := os.Open(*input)
			if e != nil {
				return &agent.Error{Code: "INPUT_UNREADABLE", HTTPStatus: 400}
			}
			defer f.Close()
			rd = f
		}
		b, e := io.ReadAll(io.LimitReader(rd, 65537))
		if e != nil || len(b) > 65536 {
			return &agent.Error{Code: "INPUT_TOO_LARGE_OR_UNREADABLE", HTTPStatus: 400}
		}
		var req agent.Request
		if agent.StrictJSON(b, &req) != nil {
			return &agent.Error{Code: "INVALID_JSON", HTTPStatus: 400}
		}
		result, _, e := engine.Query(ctx, req)
		if e != nil {
			return e
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	server, e := engine.Server(os.Getenv(cfg.Server.TokenEnv))
	if e != nil {
		return e
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case e = <-done:
		if e == http.ErrServerClosed {
			return nil
		}
		return &agent.Error{Code: "SERVER_START_FAILED", HTTPStatus: 500}
	case <-ctx.Done():
		stop, release := context.WithTimeout(context.Background(), 5*time.Second)
		defer release()
		if server.Shutdown(stop) != nil {
			return &agent.Error{Code: "SERVER_SHUTDOWN_FAILED", HTTPStatus: 500}
		}
		return nil
	}
}
