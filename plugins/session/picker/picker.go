package picker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"coding-agent/session"
)

func Select(ctx context.Context, store session.Store, r io.Reader, w io.Writer) (string, error) {
	metas, err := store.List(ctx)
	if err != nil {
		return "", err
	}
	sorted := session.SortByUpdatedDesc(metas)
	if len(sorted) == 0 {
		fmt.Fprintln(w, "no previous sessions")
		return "", nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tNAME\tUPDATED\tMSGS\tID")
	for i, m := range sorted {
		name := m.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\n",
			i+1,
			name,
			m.UpdatedAt.UTC().Format(time.RFC3339),
			m.MessageCount,
			shortID(m.ID),
		)
	}
	_ = tw.Flush()

	fmt.Fprintf(w, "\nselect session [1-%d] or q: ", len(sorted))
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" || strings.EqualFold(line, "q") {
		return "", fmt.Errorf("selection cancelled")
	}

	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err != nil || n < 1 || n > len(sorted) {
		return "", fmt.Errorf("invalid selection %q", line)
	}
	return sorted[n-1].ID, nil
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:8] + "..."
}
