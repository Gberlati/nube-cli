package oauth

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	ctx := context.Background()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url) //nolint:gosec // user-initiated browser open
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url) //nolint:gosec // user-initiated browser open
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // user-initiated browser open
	default:
		return nil
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
}
