package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

// AgentCmd groups agent-friendly helper commands.
type AgentCmd struct {
	ExitCodes AgentExitCodesCmd `cmd:"" name:"exit-codes" help:"Print stable exit code map"`
}

// AgentExitCodesCmd prints the stable exit code mapping.
type AgentExitCodesCmd struct{}

func (c *AgentExitCodesCmd) Run(ctx context.Context) error {
	if outfmt.IsJSON(ctx) {
		type entry struct {
			Code int    `json:"code"`
			Name string `json:"name"`
			Desc string `json:"description"`
		}

		entries := make([]entry, len(exitCodeMap))
		for i, e := range exitCodeMap {
			entries[i] = entry{Code: e.Code, Name: e.Name, Desc: e.Desc}
		}

		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"exit_codes": entries})
	}

	u := ui.FromContext(ctx)
	w, done := tableWriter(ctx)
	defer done()

	_, _ = fmt.Fprintln(w, "CODE\tNAME\tDESCRIPTION")

	for _, e := range exitCodeMap {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", e.Code, e.Name, e.Desc)
	}

	_ = u

	return nil
}
