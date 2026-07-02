package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"coding-agent/plan"
	"coding-agent/plugin"
	"coding-agent/session"
	"coding-agent/types"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, ag plugin.AgentHandle) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(inputPrompt(ag))

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if exitCommand(line) {
			break
		}

		if strings.HasPrefix(line, "/") {
			if handleSlashCommand(ctx, ag, line) {
				continue
			}
		}

		if err := runAgentTurn(ctx, ag, line); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}

	return nil
}

func inputPrompt(ag plugin.AgentHandle) string {
	if ag.PlanEnabled() && ag.CurrentMode() == plan.ModePlan {
		return "\n👤 you (plan)> "
	}
	return "\n👤 you> "
}

func runAgentTurn(ctx context.Context, ag plugin.AgentHandle, prompt string) error {
	fmt.Print("\n🤖 ")

	var streamCount atomic.Int32
	waitCtx, cancelWait := context.WithCancel(ctx)
	defer cancelWait()
	go showWaitingIndicator(waitCtx, &streamCount)

	onStream := func(ev types.StreamEvent) {
		streamCount.Add(1)
		fmt.Print(ev.TextDelta)
		_ = os.Stdout.Sync()
	}

	if _, err := ag.Run(ctx, prompt, onStream); err != nil {
		return err
	}

	fmt.Println()
	return nil
}

func showWaitingIndicator(ctx context.Context, streamCount *atomic.Int32) {
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
		if streamCount.Load() == 0 {
			fmt.Fprint(os.Stderr, "\n⏳ waiting for model…\n")
		}
	}
}

func handleSlashCommand(ctx context.Context, ag plugin.AgentHandle, line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case "/new":
		if err := ag.ResetSession(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Printf("new session: %s\n", ag.SessionLabel())
		}
	case "/sessions":
		metas, err := ag.ListSessions(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else if len(metas) == 0 {
			fmt.Println("no sessions")
		} else {
			printSessions(os.Stdout, metas)
		}
	case "/resume":
		if len(parts) < 2 {
			fmt.Fprintln(os.Stderr, "usage: /resume <id>")
		} else if err := ag.ResumeSession(ctx, parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Printf("resumed session: %s (mode: %s)\n", ag.SessionLabel(), ag.CurrentMode())
		}
	case "/session":
		fmt.Printf("%s (mode: %s)\n", ag.SessionLabel(), ag.CurrentMode())
	case "/name":
		if len(parts) < 2 {
			fmt.Fprintln(os.Stderr, "usage: /name <display name>")
		} else {
			name := strings.TrimSpace(strings.TrimPrefix(line, "/name"))
			if err := ag.SetSessionName(ctx, name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			} else {
				fmt.Printf("session name set: %s\n", ag.SessionLabel())
			}
		}
	case "/compact":
		instructions := strings.TrimSpace(strings.TrimPrefix(line, "/compact"))
		if err := ag.CompactSession(ctx, instructions); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Println("context compacted")
		}
	case "/plan":
		if len(parts) >= 2 && parts[1] == "show" {
			printPlan(ag)
		} else {
			prompt := strings.TrimSpace(strings.TrimPrefix(line, "/plan"))
			if err := ag.SetMode(ctx, plan.ModePlan); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			} else if prompt == "" {
				fmt.Println("plan mode (read-only)")
			} else if err := runAgentTurn(ctx, ag, prompt); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	case "/agent":
		if err := ag.SetMode(ctx, plan.ModeAgent); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Println("switched to agent mode")
		}
	case "/approve":
		prompt := strings.TrimSpace(strings.TrimPrefix(line, "/approve"))
		if err := ag.ApprovePlan(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else if err := ag.SetMode(ctx, plan.ModeAgent); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else if prompt != "" {
			if err := runAgentTurn(ctx, ag, prompt); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		} else {
			fmt.Println("plan approved; switched to agent mode")
		}
	case "/todos":
		printTodos(ag)
	default:
		fmt.Fprintln(os.Stderr, "unknown command; try /new, /sessions, /resume <id>, /session, /name <name>, /compact, /plan, /agent, /approve, /todos")
		return false
	}
	return true
}

func printPlan(ag plugin.AgentHandle) {
	if !ag.PlanEnabled() {
		fmt.Fprintln(os.Stderr, "plan mode is disabled")
		return
	}
	p := ag.CurrentPlan()
	if p == nil {
		fmt.Println("no plan")
		return
	}
	fmt.Printf("Title: %s\nStatus: %s\nOverview: %s\n", p.Title, p.Status, p.Overview)
	if id := ag.CurrentSessionID(); id != "" {
		cwd, err := os.Getwd()
		if err == nil {
			path := filepath.Join(cwd, ".coding-agent", "plans", id+".md")
			fmt.Printf("File: %s\n", path)
		}
	}
	fmt.Println()
	fmt.Println(p.Body)
}

func printTodos(ag plugin.AgentHandle) {
	todos := ag.ListTodos()
	if len(todos) == 0 {
		fmt.Println("no todos")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tCONTENT")
	for _, todo := range todos {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", todo.ID, todo.Status, todo.Content)
	}
	_ = tw.Flush()
}

func printSessions(w *os.File, metas []session.Meta) {
	sorted := session.SortByUpdatedDesc(metas)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tUPDATED\tMSGS\tMODEL")
	for _, m := range sorted {
		name := m.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
			m.ID,
			name,
			m.UpdatedAt.UTC().Format(time.RFC3339),
			m.MessageCount,
			m.Model,
		)
	}
	_ = tw.Flush()
}

func exitCommand(input string) bool {
	return input == "exit" || input == "/exit"
}

type Plugin struct{}

func (Plugin) Name() string { return "runner/repl" }

func (Plugin) Register(app *plugin.App) error {
	app.Runner = Runner{}
	return nil
}

var _ plugin.Runner = Runner{}
