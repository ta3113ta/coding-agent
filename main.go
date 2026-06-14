package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"coding-agent/agent"
	"coding-agent/tools"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ต้องตั้งค่า ANTHROPIC_API_KEY ก่อน")
		os.Exit(1)
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// ลงทะเบียน tool ทั้งหมด — เพิ่ม tool ใหม่แค่ append ตรงนี้
	registry := tools.NewRegistry(
		tools.ReadFile{},
		tools.WriteFile{},
		tools.ListDir{},
		tools.RunBash{},
	)

	ag := agent.New(client, registry, true /* verbose */)

	fmt.Println("Coding Agent (พิมพ์ 'exit' เพื่อออก)")
	fmt.Println(strings.Repeat("-", 50))

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

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
}
