package compaction

import (
	"encoding/json"
	"sort"

	"coding-agent/types"
)

type FileOps struct {
	ReadFiles     []string
	ModifiedFiles []string
}

func ExtractFileOps(msgs []types.Message, prior FileOps) FileOps {
	readSet := make(map[string]struct{}, len(prior.ReadFiles))
	modSet := make(map[string]struct{}, len(prior.ModifiedFiles))
	for _, p := range prior.ReadFiles {
		readSet[p] = struct{}{}
	}
	for _, p := range prior.ModifiedFiles {
		modSet[p] = struct{}{}
	}

	for _, m := range msgs {
		if m.Role != "assistant" {
			continue
		}
		for _, tc := range m.ToolCalls {
			switch tc.Name {
			case "read_file":
				if path := toolPath(tc.Input); path != "" {
					readSet[path] = struct{}{}
				}
			case "write_file", "str_replace":
				if path := toolPath(tc.Input); path != "" {
					modSet[path] = struct{}{}
				}
			}
		}
	}

	return FileOps{
		ReadFiles:     sortedKeys(readSet),
		ModifiedFiles: sortedKeys(modSet),
	}
}

func toolPath(input json.RawMessage) string {
	var v struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &v); err != nil {
		return ""
	}
	return v.Path
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
