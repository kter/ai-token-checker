package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kter/ai-token-checker/internal/claude"
	"github.com/kter/ai-token-checker/internal/codex"
	"github.com/kter/ai-token-checker/internal/quota"
	"github.com/kter/ai-token-checker/internal/render"
)

var version = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ai-token-checker", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit machine-readable JSON")
	noColor := flags.Bool("no-color", false, "disable ANSI colors")
	tool := flags.String("tool", "", "show only claude or codex")
	showVersion := flags.Bool("version", false, "print version and exit")
	shortHelp := flags.Bool("h", false, "show help and exit")
	showHelp := flags.Bool("help", false, "show help and exit")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: ai-token-checker [flags]")
		fmt.Fprintln(flags.Output())
		fmt.Fprintln(flags.Output(), "Print Claude Code and Codex 5-hour and weekly quota usage.")
		fmt.Fprintln(flags.Output())
		fmt.Fprintln(flags.Output(), "Flags:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", flags.Arg(0))
		flags.Usage()
		return 2
	}
	if *shortHelp || *showHelp {
		flags.Usage()
		return 0
	}
	if *showVersion {
		fmt.Fprintf(stdout, "ai-token-checker %s\n", version)
		return 0
	}
	selected := strings.ToLower(*tool)
	if selected != "" && selected != "claude" && selected != "codex" {
		fmt.Fprintln(stderr, "--tool must be claude or codex")
		return 2
	}

	now := time.Now()
	reports := make([]quota.ToolReport, 0, 2)
	if selected == "" || selected == "claude" {
		reports = append(reports, loadClaude(ctx, now))
	}
	if selected == "" || selected == "codex" {
		reports = append(reports, loadCodex())
	}

	var err error
	if *jsonOutput {
		err = render.JSON(stdout, reports, now)
	} else {
		_, noColorEnvironmentSet := os.LookupEnv("NO_COLOR")
		color := render.ColorEnabled(*noColor, render.StdoutIsTerminal(), noColorEnvironmentSet)
		err = render.Summary(stdout, reports, now, color)
	}
	if err != nil {
		fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	for _, report := range reports {
		if report.Status == "ok" {
			return 0
		}
	}
	return 1
}

func loadClaude(ctx context.Context, now time.Time) quota.ToolReport {
	token, err := claude.LoadToken(ctx)
	if err != nil {
		if errors.Is(err, claude.ErrCredentialsNotFound) {
			return quota.Failure("claude", "live", "Claude: credentials not found — skipping")
		}
		return quota.Failure("claude", "live", "Claude: unavailable — "+err.Error())
	}
	if !token.ExpiresAt.IsZero() && !token.ExpiresAt.After(now) {
		return quota.Failure("claude", "live", "Claude: token expired — run `claude` once to refresh")
	}
	windows, freshness, err := claude.NewClient().Usage(ctx, token.AccessToken)
	if err != nil {
		if errors.Is(err, claude.ErrTokenExpired) {
			return quota.Failure("claude", "live", "Claude: token expired — run `claude` once to refresh")
		}
		return quota.Failure("claude", "live", "Claude: unavailable — "+err.Error())
	}
	return quota.Success("claude", "live", windows, freshness)
}

func loadCodex() quota.ToolReport {
	home, err := os.UserHomeDir()
	if err != nil {
		return quota.Failure("codex", "local-cache", "Codex: no cached quota — run codex once")
	}
	snapshot, err := codex.LoadNewest(home)
	if err != nil {
		return quota.Failure("codex", "local-cache", "Codex: no cached quota — run codex once")
	}
	return quota.Success("codex", "local-cache", snapshot.Windows, snapshot.Freshness)
}
