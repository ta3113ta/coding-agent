package filestore

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"coding-agent/plan"
	"coding-agent/plugin"
	"coding-agent/session"
	"coding-agent/types"
)

type FileStore struct {
	dir string
}

type sessionDTO struct {
	ID          string            `json:"id"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	Name        string            `json:"name,omitempty"`
	Mode        string            `json:"mode,omitempty"`
	Todos       []plan.TodoItem   `json:"todos,omitempty"`
	Plan        *plan.Plan        `json:"plan,omitempty"`
	Messages    []messageDTO      `json:"messages"`
	Compactions []compactionDTO   `json:"compactions,omitempty"`
}

type compactionDTO struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Summary        string    `json:"summary"`
	FirstKeptIndex int       `json:"first_kept_index"`
	TokensBefore   int       `json:"tokens_before"`
	ReadFiles      []string  `json:"read_files,omitempty"`
	ModifiedFiles  []string  `json:"modified_files,omitempty"`
}

type messageDTO struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []toolCallDTO `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	IsError    bool         `json:"is_error,omitempty"`
}

type toolCallDTO struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func New(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir session dir: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

func (fs *FileStore) Create(ctx context.Context, provider, model string) (*session.Session, error) {
	_ = ctx
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	s := &session.Session{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Provider:  provider,
		Model:     model,
		Messages:  nil,
	}
	if err := fs.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (fs *FileStore) Get(ctx context.Context, id string) (*session.Session, error) {
	_ = ctx
	path, err := fs.sessionPath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %q not found", id)
		}
		return nil, fmt.Errorf("read session: %w", err)
	}
	var dto sessionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return dtoToSession(dto), nil
}

func (fs *FileStore) Save(ctx context.Context, s *session.Session) error {
	_ = ctx
	if s == nil {
		return errors.New("session is nil")
	}
	if s.CreatedAt.IsZero() {
		if existing, err := fs.Get(ctx, s.ID); err == nil {
			s.CreatedAt = existing.CreatedAt
		} else {
			s.CreatedAt = time.Now().UTC()
		}
	}
	path, err := fs.sessionPath(s.ID)
	if err != nil {
		return err
	}
	s.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(sessionToDTO(s), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write session temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename session: %w", err)
	}
	return nil
}

func (fs *FileStore) List(ctx context.Context) ([]session.Meta, error) {
	_ = ctx
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("read session dir: %w", err)
	}
	var metas []session.Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fs.dir, e.Name()))
		if err != nil {
			continue
		}
		var dto sessionDTO
		if err := json.Unmarshal(data, &dto); err != nil {
			continue
		}
		metas = append(metas, session.Meta{
			ID:           dto.ID,
			CreatedAt:    dto.CreatedAt,
			UpdatedAt:    dto.UpdatedAt,
			Provider:     dto.Provider,
			Model:        dto.Model,
			Name:         dto.Name,
			MessageCount: len(dto.Messages),
		})
	}
	return metas, nil
}

func (fs *FileStore) sessionPath(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("session id is empty")
	}
	if strings.Contains(id, "/") || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid session id: %q", id)
	}
	return filepath.Join(fs.dir, id+".json"), nil
}

func sessionToDTO(s *session.Session) sessionDTO {
	dto := sessionDTO{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Provider:  s.Provider,
		Model:     s.Model,
		Name:      s.Name,
		Mode:      s.Mode,
		Todos:     append([]plan.TodoItem(nil), s.Todos...),
		Plan:      s.Plan,
		Messages:  make([]messageDTO, len(s.Messages)),
	}
	for i, m := range s.Messages {
		dto.Messages[i] = messageDTO{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			IsError:    m.IsError,
		}
		for _, tc := range m.ToolCalls {
			dto.Messages[i].ToolCalls = append(dto.Messages[i].ToolCalls, toolCallDTO{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
		}
	}
	if len(s.Compactions) > 0 {
		dto.Compactions = make([]compactionDTO, len(s.Compactions))
		for i, c := range s.Compactions {
			dto.Compactions[i] = compactionDTO{
				ID:             c.ID,
				Timestamp:      c.Timestamp,
				Summary:        c.Summary,
				FirstKeptIndex: c.FirstKeptIndex,
				TokensBefore:   c.TokensBefore,
				ReadFiles:      append([]string(nil), c.ReadFiles...),
				ModifiedFiles:  append([]string(nil), c.ModifiedFiles...),
			}
		}
	}
	return dto
}

func dtoToSession(dto sessionDTO) *session.Session {
	s := &session.Session{
		ID:        dto.ID,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
		Provider:  dto.Provider,
		Model:     dto.Model,
		Name:      dto.Name,
		Mode:      dto.Mode,
		Todos:     append([]plan.TodoItem(nil), dto.Todos...),
		Plan:      dto.Plan,
		Messages:  make([]types.Message, len(dto.Messages)),
	}
	for i, m := range dto.Messages {
		s.Messages[i] = types.Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			IsError:    m.IsError,
		}
		for _, tc := range m.ToolCalls {
			s.Messages[i].ToolCalls = append(s.Messages[i].ToolCalls, types.ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
		}
	}
	if len(dto.Compactions) > 0 {
		s.Compactions = make([]session.CompactionRecord, len(dto.Compactions))
		for i, c := range dto.Compactions {
			s.Compactions[i] = session.CompactionRecord{
				ID:             c.ID,
				Timestamp:      c.Timestamp,
				Summary:        c.Summary,
				FirstKeptIndex: c.FirstKeptIndex,
				TokensBefore:   c.TokensBefore,
				ReadFiles:      append([]string(nil), c.ReadFiles...),
				ModifiedFiles:  append([]string(nil), c.ModifiedFiles...),
			}
		}
	}
	return s
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "session/filestore" }

func (Plugin) Register(app *plugin.App) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	dir, err := app.Config.SessionDirPath(cwd)
	if err != nil {
		return err
	}
	store, err := New(dir)
	if err != nil {
		return err
	}
	app.SessionStore = store
	return nil
}

var _ session.Store = (*FileStore)(nil)
