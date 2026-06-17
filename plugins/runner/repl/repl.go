package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"coding-agent/plugin"
	"coding-agent/types"
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

		fmt.Print("\n🤖 ")
		_, runErr := ag.Run(ctx, line, func(ev types.StreamEvent) {
			fmt.Print(ev.TextDelta)
		})
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
			continue
		}
		fmt.Println()
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
