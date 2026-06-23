package agent

import (
	"context"
	"fmt"
	"strings"

	"coding-agent/compaction"
	"coding-agent/llm"
	"coding-agent/permission"
	"coding-agent/session"
	"coding-agent/tools"
	"coding-agent/types"
)

type Agent struct {
	provider     llm.Provider
	registry     *tools.Registry
	archive      []types.Message
	compactions  []session.CompactionRecord
	messages     []types.Message
	model        string
	systemPrompt string
	promptCache  types.PromptCacheConfig
	verbose      bool
	store        session.Store
	sessionID    string
	sessionName  string
	providerName string
	permission   *permission.Chain
	compactor    compaction.Compactor
}

func New(provider llm.Provider, registry *tools.Registry, model, systemPrompt string, promptCache types.PromptCacheConfig, verbose bool, store session.Store, providerName string, perm *permission.Chain, compactor compaction.Compactor) (*Agent, error) {
	if store == nil {
		return nil, fmt.Errorf("session store is required")
	}
	return &Agent{
		provider:     provider,
		registry:     registry,
		model:        model,
		systemPrompt: systemPrompt,
		promptCache:  promptCache,
		verbose:      verbose,
		store:        store,
		providerName: providerName,
		permission:   perm,
		compactor:    compactor,
	}, nil
}

func (a *Agent) CurrentSessionID() string {
	return a.sessionID
}

func (a *Agent) CurrentSessionName() string {
	return a.sessionName
}

func (a *Agent) SessionLabel() string {
	if a.sessionName != "" {
		return fmt.Sprintf("%s (%s)", a.sessionName, a.sessionID)
	}
	return a.sessionID
}

func (a *Agent) ListSessions(ctx context.Context) ([]session.Meta, error) {
	return a.store.List(ctx)
}

func (a *Agent) InitNewSession(ctx context.Context) error {
	s, err := a.store.Create(ctx, a.providerName, a.model)
	if err != nil {
		return err
	}
	a.sessionID = s.ID
	a.sessionName = ""
	a.archive = nil
	a.compactions = nil
	a.messages = nil
	return nil
}

func (a *Agent) ResumeSession(ctx context.Context, id string) error {
	s, err := a.store.Get(ctx, id)
	if err != nil {
		return err
	}
	a.sessionID = s.ID
	a.sessionName = s.Name
	a.archive = append([]types.Message(nil), s.Messages...)
	a.compactions = append([]session.CompactionRecord(nil), s.Compactions...)
	a.rebuildProjection()
	return a.maybeCompact(ctx, false, "")
}

func (a *Agent) CompactSession(ctx context.Context, customInstructions string) error {
	if a.sessionID == "" {
		return fmt.Errorf("no active session")
	}
	if err := a.maybeCompact(ctx, true, customInstructions); err != nil {
		return err
	}
	return a.persist(ctx)
}

func (a *Agent) ResetSession(ctx context.Context) error {
	return a.InitNewSession(ctx)
}

func (a *Agent) SetSessionName(ctx context.Context, name string) error {
	if a.sessionID == "" {
		return fmt.Errorf("no active session")
	}
	a.sessionName = strings.TrimSpace(name)
	return a.persist(ctx)
}

func (a *Agent) persist(ctx context.Context) error {
	return a.store.Save(ctx, &session.Session{
		ID:          a.sessionID,
		Name:        a.sessionName,
		Provider:    a.providerName,
		Model:       a.model,
		Messages:    a.archive,
		Compactions: a.compactions,
	})
}

func (a *Agent) rebuildProjection() {
	a.messages = compaction.ProjectMessages(a.archive, a.compactions)
}

func (a *Agent) appendArchive(msgs ...types.Message) {
	a.archive = append(a.archive, msgs...)
	a.rebuildProjection()
}

func (a *Agent) maybeCompact(ctx context.Context, force bool, customInstructions string) error {
	if a.compactor == nil {
		return nil
	}
	res, err := a.compactor.MaybeCompact(ctx, compaction.Request{
		Archive:            a.archive,
		Compactions:        a.compactions,
		SystemPrompt:       a.systemPrompt,
		Model:              a.model,
		Force:              force,
		CustomInstructions: customInstructions,
	})
	if err != nil {
		return fmt.Errorf("compact: %w", err)
	}
	a.archive = res.Archive
	a.compactions = res.Compactions
	a.messages = res.Projected
	return nil
}

func (a *Agent) Run(ctx context.Context, userInput string, onStream func(types.StreamEvent)) (string, error) {
	if a.sessionID == "" {
		return "", fmt.Errorf("no active session")
	}

	a.appendArchive(types.Message{
		Role:    "user",
		Content: userInput,
	})

	for {
		if err := a.maybeCompact(ctx, false, ""); err != nil {
			return "", err
		}

		resp, err := a.provider.Complete(ctx, types.CompleteRequest{
			SystemPrompt: a.systemPrompt,
			Messages:     a.messages,
			Tools:        a.registry.Definitions(),
			Model:        a.model,
			MaxTokens:    8096,
			OnStream:     onStream,
			PromptCache:  a.promptCache,
		})
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		a.appendArchive(types.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			if err := a.persist(ctx); err != nil {
				return "", fmt.Errorf("save session: %w", err)
			}
			return resp.Text, nil
		}

		if a.verbose && resp.Text != "" {
			fmt.Printf("\n💭 %s\n", resp.Text)
		}

		for _, tc := range resp.ToolCalls {
			if a.verbose {
				fmt.Printf("🔧 %s(%s)\n", tc.Name, string(tc.Input))
			}

			input := tc.Input
			if a.permission != nil && !a.permission.Empty() {
				permRes, err := a.permission.Evaluate(ctx, permission.ToolUseRequest{
					ToolName:   tc.Name,
					Input:      tc.Input,
					ToolCallID: tc.ID,
				})
				if err != nil {
					a.appendArchive(types.Message{
						Role:       "tool",
						Content:    fmt.Sprintf("error: permission check failed: %v", err),
						ToolCallID: tc.ID,
						IsError:    true,
					})
					continue
				}
				if permRes.Decision == permission.Deny {
					msg := permRes.Message
					if msg == "" {
						msg = "permission denied"
					}
					a.appendArchive(types.Message{
						Role:       "tool",
						Content:    fmt.Sprintf("error: %s", msg),
						ToolCallID: tc.ID,
						IsError:    true,
					})
					continue
				}
				if permRes.UpdatedInput != nil {
					input = permRes.UpdatedInput
				}
			}

			result, err := a.registry.Dispatch(tc.Name, input)
			isError := false
			if err != nil {
				result = fmt.Sprintf("error: %v", err)
				isError = true
			}

			a.appendArchive(types.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				IsError:    isError,
			})
		}
	}
}
