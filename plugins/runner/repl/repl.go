package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"coding-agent/plugin"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, ag plugin.AgentHandle) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n👤 hey> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}

		answer, err := ag.Run(ctx, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Printf("\n🤖 %s\n", answer)
	}
	return nil
}

type Plugin struct{}

func (Plugin) Name() string { return "runner/repl" }

func (Plugin) Register(app *plugin.App) error {
	app.Runner = Runner{}
	return nil
}

var _ plugin.Runner = Runner{}
