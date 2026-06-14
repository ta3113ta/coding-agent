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
)

func main() {
	providerFlag := flag.String("provider", "", "LLM provider: anthropic|openrouter")
	modelFlag := flag.String("model", "", "Model override")
	flag.Parse()

	cfg := plugin.LoadConfigFromEnv()
	cfg.ApplyFlags(*providerFlag, *modelFlag)

	app, err := plugin.Bootstrap(cfg, builtin.Default...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ag := agent.New(app.Provider, app.Tools, cfg.Model(), app.Prompt, true /* verbose */)

	fmt.Printf("Coding Agent [%s / %s] (พิมพ์ 'exit' เพื่อออก)\n", cfg.Provider, cfg.Model())
	fmt.Println(strings.Repeat("-", 50))

	if err := app.Runner.Run(context.Background(), ag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
