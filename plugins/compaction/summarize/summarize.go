package summarize

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"coding-agent/compaction"
	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/plugin"
	"coding-agent/session"
	"coding-agent/types"
)

type Compactor struct {
	provider llm.Provider
	cfg      config.Config
	verbose  bool
}

func (c *Compactor) MaybeCompact(ctx context.Context, req compaction.Request) (compaction.Result, error) {
	archive := req.Archive
	compactions := req.Compactions
	projected := compaction.ProjectMessages(archive, compactions)

	if len(archive) == 0 {
		return compaction.Result{
			Archive:     archive,
			Compactions: compactions,
			Projected:   projected,
		}, nil
	}

	contextWindow := compaction.ContextWindowForModel(req.Model, c.cfg.CompactionContextWindow)
	if req.ContextWindow > 0 {
		contextWindow = req.ContextWindow
	}

	estimate := compaction.EstimateTokens(projected)
	if !req.Force && !compaction.ShouldCompact(estimate, contextWindow, c.cfg.CompactionReserveTokens) {
		return compaction.Result{
			Archive:     archive,
			Compactions: compactions,
			Projected:   projected,
		}, nil
	}

	_, suffix := compaction.SplitMessagesByTokens(archive, c.cfg.CompactionKeepRecentTokens)
	cutIndex := len(archive) - len(suffix)
	if cutIndex <= 0 && len(archive) > 1 {
		_, suffix = compaction.SplitMessages(archive, 1)
		cutIndex = len(archive) - len(suffix)
	}
	if cutIndex <= 0 {
		return compaction.Result{
			Archive:     archive,
			Compactions: compactions,
			Projected:   projected,
		}, nil
	}

	summarizeFrom := 0
	var previousSummary string
	var priorOps compaction.FileOps
	if len(compactions) > 0 {
		last := compactions[len(compactions)-1]
		summarizeFrom = last.FirstKeptIndex
		previousSummary = last.Summary
		priorOps = compaction.FileOps{
			ReadFiles:     append([]string(nil), last.ReadFiles...),
			ModifiedFiles: append([]string(nil), last.ModifiedFiles...),
		}
	}

	toSummarize := archive[summarizeFrom:cutIndex]
	if len(toSummarize) == 0 {
		return compaction.Result{
			Archive:     archive,
			Compactions: compactions,
			Projected:   projected,
		}, nil
	}

	fileOps := compaction.ExtractFileOps(archive[:cutIndex], priorOps)
	summary, err := c.summarize(ctx, req, toSummarize, previousSummary, fileOps)
	if err != nil {
		return compaction.Result{}, fmt.Errorf("summarize: %w", err)
	}

	recordID, err := newCompactionID()
	if err != nil {
		return compaction.Result{}, err
	}

	record := session.CompactionRecord{
		ID:             recordID,
		Timestamp:      time.Now().UTC(),
		Summary:        summary,
		FirstKeptIndex: cutIndex,
		TokensBefore:   estimate,
		ReadFiles:      fileOps.ReadFiles,
		ModifiedFiles:  fileOps.ModifiedFiles,
	}
	newCompactions := append(append([]session.CompactionRecord(nil), compactions...), record)
	newProjected := compaction.ProjectMessages(archive, newCompactions)

	if c.verbose {
		fmt.Printf("📦 compacted archive %d msgs (kept from %d, est. %d → %d tokens)\n",
			len(archive), cutIndex, estimate, compaction.EstimateTokens(newProjected))
	}

	return compaction.Result{
		Archive:     archive,
		Compactions: newCompactions,
		Projected:   newProjected,
		Compacted:   true,
	}, nil
}

func (c *Compactor) summarize(ctx context.Context, req compaction.Request, msgs []types.Message, previousSummary string, fileOps compaction.FileOps) (string, error) {
	serialized := compaction.SerializeConversation(msgs, compaction.SerializeOptions{})
	prompt := compaction.BuildSummaryPrompt(serialized, previousSummary, fileOps, req.CustomInstructions)

	resp, err := c.provider.Complete(ctx, types.CompleteRequest{
		SystemPrompt: req.SystemPrompt,
		Messages: []types.Message{
			{Role: "user", Content: prompt},
		},
		Model:     req.Model,
		MaxTokens: 4096,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resp.Text) == "" {
		return "", fmt.Errorf("empty summary from provider")
	}
	return resp.Text, nil
}

func newCompactionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random compaction id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "compaction/summarize" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.CompactionEnabled {
		return nil
	}
	provider, err := llm.NewProvider(app.Config)
	if err != nil {
		return err
	}
	plugin.RegisterCompactor(app, &Compactor{
		provider: provider,
		cfg:      app.Config,
		verbose:  true,
	})
	return nil
}
