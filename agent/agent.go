package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"coding-agent/compaction"
	"coding-agent/llm"
	"coding-agent/permission"
	"coding-agent/plan"
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
	planState    *plan.SessionState
	planEnabled  bool
}

var (
	errSessionStoreRequired = errors.New("session store is required")
	errPlanModeDisabled     = errors.New("plan mode is disabled")
	errNoActiveSession      = errors.New("no active session")
	errPlanStateUnavailable = errors.New("plan state unavailable")
)

func New(provider llm.Provider, registry *tools.Registry, model, systemPrompt string, promptCache types.PromptCacheConfig, verbose bool, store session.Store, providerName string, perm *permission.Chain, compactor compaction.Compactor, planState *plan.SessionState, planEnabled bool) (*Agent, error) {
	if store == nil {
		return nil, errSessionStoreRequired
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
		planState:    planState,
		planEnabled:  planEnabled,
	}, nil
}

func (a *Agent) PlanEnabled() bool {
	return a.planEnabled
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

func (a *Agent) CurrentMode() plan.Mode {
	if a.planState == nil {
		return plan.ModeAgent
	}
	return a.planState.Mode()
}

func (a *Agent) ListTodos() []plan.TodoItem {
	if a.planState == nil {
		return nil
	}
	return a.planState.Todos()
}

func (a *Agent) CurrentPlan() *plan.Plan {
	if a.planState == nil {
		return nil
	}
	return a.planState.Plan()
}

func (a *Agent) CanSwitchToAgent() (bool, string) {
	if a.planState == nil {
		return true, ""
	}
	return a.planState.CanSwitchToAgent()
}

func (a *Agent) SetMode(ctx context.Context, mode plan.Mode) error {
	if !a.planEnabled {
		return errPlanModeDisabled
	}
	if a.sessionID == "" {
		return errNoActiveSession
	}
	if a.planState == nil {
		return errPlanStateUnavailable
	}
	if mode == plan.ModeAgent {
		ok, msg := a.planState.CanSwitchToAgent()
		if !ok {
			return errors.New(msg)
		}
	}
	a.planState.SetMode(mode)
	return a.persist(ctx)
}

func (a *Agent) ApprovePlan(ctx context.Context) error {
	if !a.planEnabled {
		return errPlanModeDisabled
	}
	if a.sessionID == "" {
		return errNoActiveSession
	}
	if a.planState == nil {
		return errPlanStateUnavailable
	}
	if err := a.planState.ApprovePlan(); err != nil {
		return err
	}
	return a.persist(ctx)
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
	if a.planState != nil {
		a.planState.SetSessionID(s.ID)
		a.planState.LoadSnapshot(plan.Snapshot{
			Mode:  s.Mode,
			Todos: s.Todos,
			Plan:  s.Plan,
		})
	}
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
	if a.planState != nil {
		a.planState.SetSessionID(s.ID)
		a.planState.LoadSnapshot(plan.Snapshot{
			Mode:  s.Mode,
			Todos: s.Todos,
			Plan:  s.Plan,
		})
	}
	a.rebuildProjection()
	if a.permission != nil {
		a.permission.ClearSessionRules()
	}
	return a.maybeCompact(ctx, false, "")
}

func (a *Agent) CompactSession(ctx context.Context, customInstructions string) error {
	if a.sessionID == "" {
		return errNoActiveSession
	}
	if err := a.maybeCompact(ctx, true, customInstructions); err != nil {
		return err
	}
	return a.persist(ctx)
}

func (a *Agent) ResetSession(ctx context.Context) error {
	if err := a.InitNewSession(ctx); err != nil {
		return err
	}
	if a.permission != nil {
		a.permission.ClearSessionRules()
	}
	return nil
}

func (a *Agent) SetSessionName(ctx context.Context, name string) error {
	if a.sessionID == "" {
		return errNoActiveSession
	}
	a.sessionName = strings.TrimSpace(name)
	return a.persist(ctx)
}

func (a *Agent) persist(ctx context.Context) error {
	sess := &session.Session{
		ID:          a.sessionID,
		Name:        a.sessionName,
		Provider:    a.providerName,
		Model:       a.model,
		Messages:    a.archive,
		Compactions: a.compactions,
	}
	if a.planState != nil {
		snap := a.planState.Snapshot()
		sess.Mode = snap.Mode
		sess.Todos = snap.Todos
		sess.Plan = snap.Plan
	}
	return a.store.Save(ctx, sess)
}

func (a *Agent) rebuildProjection() {
	a.messages = compaction.ProjectMessages(a.archive, a.compactions)
}

func (a *Agent) appendArchive(msgs ...types.Message) {
	a.archive = append(a.archive, msgs...)
	a.rebuildProjection()
}

func (a *Agent) effectiveSystemPrompt() string {
	if a.planEnabled && a.planState != nil && a.planState.Mode() == plan.ModePlan {
		return a.systemPrompt + "\n\n" + plan.PlanModePromptSuffix
	}
	return a.systemPrompt
}

func (a *Agent) toolDefinitions() []types.ToolDefinition {
	if a.planEnabled && a.planState != nil && a.planState.Mode() == plan.ModePlan {
		return a.registry.DefinitionsFiltered(plan.AllowedToolsInPlanMode())
	}
	return a.registry.Definitions()
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
		SessionID:          a.sessionID,
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
		return "", errNoActiveSession
	}

	a.appendArchive(types.Message{
		Role:    "user",
		Content: userInput,
	})

	text, err := a.runLoop(ctx, 0, onStream)
	if err != nil {
		return "", err
	}
	if err := a.persist(ctx); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}
	return text, nil
}

// RunSubtask runs an isolated turn sequence on the current session (ephemeral child agents).
// maxTurns limits LLM rounds; 0 means unlimited. When exceeded, returns last text with a suffix.
func (a *Agent) RunSubtask(ctx context.Context, prompt string, maxTurns int) (string, error) {
	if a.sessionID == "" {
		return "", errNoActiveSession
	}

	a.appendArchive(types.Message{
		Role:    "user",
		Content: prompt,
	})

	text, err := a.runLoop(ctx, maxTurns, nil)
	if err != nil {
		return "", err
	}
	if err := a.persist(ctx); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}
	return text, nil
}

func (a *Agent) runLoop(ctx context.Context, maxTurns int, onStream func(types.StreamEvent)) (string, error) {
	turns := 0
	var lastText string

	for {
		if maxTurns > 0 && turns >= maxTurns {
			if lastText == "" {
				lastText = "[sub-agent stopped: max turns reached]"
			} else {
				lastText += "\n\n[sub-agent stopped: max turns reached]"
			}
			return lastText, nil
		}
		turns++

		if err := a.maybeCompact(ctx, false, ""); err != nil {
			return "", err
		}

		const maxEmptyRetries = 2
		var resp *types.CompleteResponse
		var err error
		for attempt := 0; attempt <= maxEmptyRetries; attempt++ {
			resp, err = a.provider.Complete(ctx, types.CompleteRequest{
				SystemPrompt: a.effectiveSystemPrompt(),
				Messages:     a.messages,
				Tools:        a.toolDefinitions(),
				Model:        a.model,
				MaxTokens:    8096,
				OnStream:     onStream,
				PromptCache:  a.promptCache,
				SessionID:    a.sessionID,
			})
			if err != nil {
				if attempt < maxEmptyRetries && isRetryableLLMError(err) {
					continue
				}
				break
			}
			if strings.TrimSpace(resp.Text) != "" || len(resp.ToolCalls) > 0 {
				break
			}
			err = errors.New("llm call: model returned empty response")
			if attempt < maxEmptyRetries {
				continue
			}
		}

		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		lastText = resp.Text

		a.appendArchive(types.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			return resp.Text, nil
		}

		if a.verbose && resp.Text != "" {
			fmt.Printf("\n💭 %s\n", resp.Text)
		}

		for _, tc := range resp.ToolCalls {
			if a.verbose {
				fmt.Printf("🔧 %s(%s)\n", tc.Name, string(tc.Input))
			}

			if a.planEnabled && a.planState != nil && a.planState.Mode() == plan.ModePlan && !plan.IsAllowedInPlanMode(tc.Name) {
				a.appendArchive(types.Message{
					Role:       "tool",
					Content:    fmt.Sprintf("error: tool %q is not allowed in plan mode", tc.Name),
					ToolCallID: tc.ID,
					IsError:    true,
				})
				continue
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

			result, err := a.registry.Dispatch(ctx, tc.Name, input)
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

func isRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "empty response") ||
		strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "deadline exceeded")
}
