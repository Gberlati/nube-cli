package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/alecthomas/kong"

	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

// SchemaCmd emits a machine-readable schema of all commands and flags.
type SchemaCmd struct{}

func (c *SchemaCmd) Run(ctx context.Context) error {
	parser, _, err := newParser(baseDescription())
	if err != nil {
		return err
	}

	schema := buildSchema(parser.Model.Node)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, schema)
	}

	// Plain: just output compact JSON since it's machine-oriented.
	u := ui.FromContext(ctx)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(schema); err != nil {
		return err
	}

	_ = u

	return nil
}

func buildSchema(node *kong.Node) map[string]any {
	result := map[string]any{
		"name": node.Name,
	}

	if node.Help != "" {
		result["help"] = node.Help
	}

	if len(node.Aliases) > 0 {
		result["aliases"] = node.Aliases
	}

	if flags := schemaFlags(node); len(flags) > 0 {
		result["flags"] = flags
	}

	if args := schemaArgs(node); len(args) > 0 {
		result["args"] = args
	}

	if children := schemaChildren(node); len(children) > 0 {
		result["commands"] = children
	}

	return result
}

func schemaFlags(node *kong.Node) []map[string]any {
	var flags []map[string]any

	for _, f := range node.Flags {
		if f.Hidden || f.Name == "help" {
			continue
		}

		entry := map[string]any{
			"name": "--" + f.Name,
			"type": f.Value.Target.Type().String(),
		}

		if f.Help != "" {
			entry["help"] = f.Help
		}

		if f.Short != 0 {
			entry["short"] = "-" + string(f.Short)
		}

		if f.HasDefault {
			entry["default"] = f.Default
		}

		if len(f.Aliases) > 0 {
			entry["aliases"] = f.Aliases
		}

		if f.Required {
			entry["required"] = true
		}

		flags = append(flags, entry)
	}

	return flags
}

func schemaArgs(node *kong.Node) []map[string]any {
	args := make([]map[string]any, 0, len(node.Positional))

	for _, a := range node.Positional {
		entry := map[string]any{
			"name": a.Name,
			"type": a.Target.Type().String(),
		}

		if a.Help != "" {
			entry["help"] = a.Help
		}

		if a.Required {
			entry["required"] = true
		}

		args = append(args, entry)
	}

	return args
}

func schemaChildren(node *kong.Node) []map[string]any {
	var children []map[string]any

	for _, child := range node.Children {
		if child == nil || child.Hidden {
			continue
		}

		children = append(children, buildSchema(child))
	}

	return children
}
