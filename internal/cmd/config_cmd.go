package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gberlati/nube-cli/internal/config"
	"github.com/gberlati/nube-cli/internal/credstore"
	"github.com/gberlati/nube-cli/internal/outfmt"
)

type ConfigCmd struct {
	List ConfigListCmd `cmd:"" aliases:"ls,all" default:"withargs" help:"List all config values"`
	Path ConfigPathCmd `cmd:"" aliases:"where" help:"Print config file path"`
}

type ConfigListCmd struct{}

func (c *ConfigListCmd) Run(ctx context.Context) error {
	path, _ := config.ConfigPath()
	credPath, _ := credstore.Path()

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"config_path":      path,
			"credentials_path": credPath,
		})
	}

	fmt.Fprintf(os.Stdout, "Config file: %s\n", path)
	fmt.Fprintf(os.Stdout, "Credentials: %s\n", credPath)

	return nil
}

type ConfigPathCmd struct{}

func (c *ConfigPathCmd) Run(ctx context.Context) error {
	path, err := config.ConfigPath()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, outfmt.PathPayload(path))
	}

	fmt.Fprintln(os.Stdout, path)

	return nil
}
