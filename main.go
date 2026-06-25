package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"coding-agent/agent"
	"coding-agent/plugin"
	"coding-agent/plugins/builtin"
	"coding-agent/plugins/session/memory"
	"coding-agent/plugins/session/picker"
	"coding-agent/session"
)

type startupFlags struct {
	continueLatest bool
	pickSession    bool
	noSession      bool
	newSession     bool
	resume         string
	name           string
}

func main() {
	providerFlag := flag.String("provider", "", "LLM provider: anthropic|openrouter")
	modelFlag := flag.String("model", "", "Model override")
	sessionScopeFlag := flag.String("session-scope", "", "Session storage scope: project|global")
	sessionDirFlag := flag.String("session-dir", "", "Override session storage directory")
	resumeFlag := flag.String("resume", "", "Resume session ID on startup")
	newSessionFlag := flag.Bool("new-session", false, "Force new session (overrides --resume)")
	continueFlag := flag.Bool("c", false, "Continue most recent session")
	pickFlag := flag.Bool("r", false, "Browse and select a past session")
	noSessionFlag := flag.Bool("no-session", false, "Ephemeral mode; do not save sessions")
	noPermissionFlag := flag.Bool("no-permission", false, "Disable permission hooks before tool execution")
	noCompactionFlag := flag.Bool("no-compaction", false, "Disable context compaction")
	noSpawnFlag := flag.Bool("no-spawn", false, "Disable sub-agent task spawning")
	nameFlag := flag.String("name", "", "Set session display name at startup")
	flag.Parse()

	flags := startupFlags{
		continueLatest: *continueFlag,
		pickSession:    *pickFlag,
		noSession:      *noSessionFlag,
		newSession:     *newSessionFlag,
		resume:         strings.TrimSpace(*resumeFlag),
		name:           strings.TrimSpace(*nameFlag),
	}

	if err := validateStartupFlags(flags); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := plugin.LoadConfigFromEnv()
	cfg.ApplyFlags(*providerFlag, *modelFlag)
	cfg.ApplySessionFlags(*sessionScopeFlag, *sessionDirFlag)
	cfg.ApplyPermissionFlags(*noPermissionFlag)
	cfg.ApplyCompactionFlags(*noCompactionFlag)
	cfg.ApplySpawnFlags(*noSpawnFlag)

	app, err := plugin.Bootstrap(cfg, builtin.Default...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := app.SessionStore
	if flags.noSession {
		store = memory.New()
	}

	if store == nil {
		fmt.Fprintln(os.Stderr, "session store not configured")
		os.Exit(1)
	}

	// FIXME: less parameters to the agent constructor
	ag, err := agent.New(
		app.Provider,
		app.Tools,
		cfg.Model(),
		app.Prompt,
		cfg.PromptCache(),
		true, /* verbose */
		store,
		string(cfg.Provider),
		app.Permission,
		app.Compactor,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := resolveStartupSession(ctx, ag, store, flags); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if flags.name != "" {
		if err := ag.SetSessionName(ctx, flags.name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Coding Agent [%s / %s] (type 'exit' to quit)\n", cfg.Provider, cfg.Model())
	fmt.Printf("Session: %s\n", ag.SessionLabel())
	fmt.Println(strings.Repeat("-", 50))

	if err := app.Runner.Run(ctx, ag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateStartupFlags(f startupFlags) error {
	if f.noSession {
		if f.continueLatest || f.pickSession || f.newSession || f.resume != "" {
			return fmt.Errorf("--no-session cannot be combined with -c, -r, --new-session, or --resume")
		}
		return nil
	}
	if f.newSession && (f.continueLatest || f.pickSession || f.resume != "") {
		return fmt.Errorf("--new-session cannot be combined with -c, -r, or --resume")
	}
	if f.resume != "" && (f.continueLatest || f.pickSession) {
		return fmt.Errorf("--resume cannot be combined with -c or -r")
	}
	if f.continueLatest && f.pickSession {
		return fmt.Errorf("-c and -r are mutually exclusive")
	}
	return nil
}

func resolveStartupSession(ctx context.Context, ag *agent.Agent, store session.Store, f startupFlags) error {
	switch {
	case f.newSession || (!f.continueLatest && !f.pickSession && f.resume == ""):
		return ag.InitNewSession(ctx)
	case f.resume != "":
		return ag.ResumeSession(ctx, f.resume)
	case f.pickSession:
		id, err := picker.Select(ctx, store, os.Stdin, os.Stdout)
		if err != nil {
			return err
		}
		if id == "" {
			return ag.InitNewSession(ctx)
		}
		return ag.ResumeSession(ctx, id)
	case f.continueLatest:
		metas, err := store.List(ctx)
		if err != nil {
			return err
		}
		if latest := session.Latest(metas); latest != nil {
			return ag.ResumeSession(ctx, latest.ID)
		}
		fmt.Println("no previous session; starting fresh")
		return ag.InitNewSession(ctx)
	default:
		return ag.InitNewSession(ctx)
	}
}
