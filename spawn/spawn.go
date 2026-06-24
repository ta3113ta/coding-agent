package spawn

import "context"

type Type string

const (
	TypeGeneralPurpose Type = "generalPurpose"
	TypeExplore        Type = "explore"
	TypeShell          Type = "shell"
)

type Request struct {
	Type        Type
	Description string
	Prompt      string
}

type Result struct {
	Text string
}

type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}
