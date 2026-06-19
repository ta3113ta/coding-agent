package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"coding-agent/plugin"
	"coding-agent/session"
	"coding-agent/types"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, ag plugin.AgentHandle) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n👤 you> ")

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

		fmt.Print("\n🤖 ")

		onStream := func(ev types.StreamEvent) {
			fmt.Print(ev.TextDelta)
		}

		_, runErr := ag.Run(ctx, line, onStream)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
			continue
		}

		fmt.Println()
	}

	return nil
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
			fmt.Printf("resumed session: %s\n", ag.SessionLabel())
		}
	case "/session":
		fmt.Println(ag.SessionLabel())
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
	default:
		fmt.Fprintln(os.Stderr, "unknown command; try /new, /sessions, /resume <id>, /session, /name <name>")
		return false
	}
	return true
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
