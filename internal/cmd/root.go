package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/gberlati/nube-cli/internal/credstore"
	"github.com/gberlati/nube-cli/internal/errfmt"
	"github.com/gberlati/nube-cli/internal/outfmt"
	"github.com/gberlati/nube-cli/internal/ui"
)

const (
	colorAuto  = "auto"
	colorNever = "never"
)

type RootFlags struct {
	Color          string `help:"Color output: auto|always|never" default:"${color}"`
	Store          string `help:"Store profile name" short:"s" env:"NUBE_STORE"`
	EnableCommands string `help:"Comma-separated list of enabled top-level commands (restricts CLI)" default:"${enabled_commands}"`
	JSON           bool   `help:"Output JSON to stdout (best for scripting)" default:"${json}" short:"j"`
	Plain          bool   `help:"Output stable, parseable text to stdout (TSV; no colors)" default:"${plain}" short:"p"`
	Select         string `help:"Comma-separated list of fields to select from JSON output (supports dot paths)" short:"S"`
	Force          bool   `help:"Skip confirmations for destructive commands" aliases:"yes,assume-yes" short:"y"`
	NoInput        bool   `help:"Never prompt; fail instead (useful for CI)" aliases:"non-interactive,noninteractive"`
	DryRun         bool   `help:"Show what would be done without executing" short:"n"`
	Verbose        bool   `help:"Enable verbose logging" short:"v"`
}

type CLI struct {
	RootFlags `embed:""`

	Version kong.VersionFlag `help:"Print version and exit"`

	// Desire paths — agent-friendly shortcuts.
	Shop     StoreGetCmd    `cmd:"" name:"shop" help:"Show store info (alias for 'store get')"`
	Products ProductListCmd `cmd:"" name:"products" help:"List products (alias for 'product list')"`
	Orders   OrderListCmd   `cmd:"" name:"orders" help:"List orders (alias for 'order list')"`
	Status   AuthStatusCmd  `cmd:"" name:"status" help:"Show auth status (alias for 'auth status')"`
	Login    LoginCmd       `cmd:"" name:"login" help:"Authorize and store a profile"`
	Logout   LogoutCmd      `cmd:"" name:"logout" help:"Remove a store profile"`

	// Domain commands.
	Auth     AuthCmd     `cmd:"" help:"Auth and credentials"`
	Store    StoreCmd    `cmd:"" help:"Store information"`
	Product  ProductCmd  `cmd:"" aliases:"prod" help:"Manage products"`
	Order    OrderCmd    `cmd:"" aliases:"ord" help:"Manage orders"`
	Category CategoryCmd `cmd:"" aliases:"cat" help:"Manage categories"`
	Customer CustomerCmd `cmd:"" aliases:"cust" help:"Manage customers"`
	Config   ConfigCmd   `cmd:"" help:"Manage configuration"`
	Agent    AgentCmd    `cmd:"" help:"Agent-friendly helpers"`
	Schema   SchemaCmd   `cmd:"" help:"Machine-readable command schema" aliases:"help-json"`

	VersionCmd VersionCmd `cmd:"" name:"version" help:"Print version"`
	Help       HelpCmd    `cmd:"" help:"Show help (same as --help)"`
}

type exitPanic struct{ code int }

func Execute(args []string) (err error) {
	parser, cli, err := newParser(helpDescription())
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				if ep.code == 0 {
					err = nil
					return
				}

				err = &ExitErr{Code: ep.code, Err: errors.New("exited")}

				return
			}

			panic(r)
		}
	}()

	kctx, err := parser.Parse(args)
	if err != nil {
		parsedErr := wrapParseError(err)
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(parsedErr))

		return parsedErr
	}

	if err = enforceEnabledCommands(kctx, cli.EnableCommands); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
		return err
	}

	logLevel := slog.LevelWarn
	if cli.Verbose {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	mode, err := outfmt.FromFlags(cli.JSON, cli.Plain)
	if err != nil {
		return newUsageError(err)
	}

	ctx := context.Background()
	ctx = outfmt.WithMode(ctx, mode)

	if cli.Select != "" {
		fields := strings.Split(cli.Select, ",")
		ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{Select: fields})
	}

	uiColor := cli.Color
	if outfmt.IsJSON(ctx) || outfmt.IsPlain(ctx) {
		uiColor = colorNever
	}

	u, err := ui.New(ui.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Color:  uiColor,
	})
	if err != nil {
		return err
	}

	ctx = ui.WithUI(ctx, u)

	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.Bind(&cli.RootFlags)
	kctx.Bind(parser)

	err = kctx.Run()
	if err == nil {
		return nil
	}

	if ExitCode(err) == 0 {
		return nil
	}

	// Wrap with stable exit code if not already wrapped.
	var ee *ExitErr
	if !errors.As(err, &ee) {
		err = &ExitErr{Code: stableExitCode(err), Err: err}
	}

	if u := ui.FromContext(ctx); u != nil {
		msg := strings.TrimSpace(errfmt.Format(err))
		if msg != "" {
			u.Err().Error(msg)
		}

		return err
	}

	msg := strings.TrimSpace(errfmt.Format(err))
	if msg != "" {
		_, _ = fmt.Fprintln(os.Stderr, msg)
	}

	return err
}

func wrapParseError(err error) error {
	if err == nil {
		return nil
	}

	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return &ExitErr{Code: ExitUsage, Err: parseErr}
	}

	return err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func boolString(v bool) string {
	return strconv.FormatBool(v)
}

func usagef(format string, args ...any) error {
	return &ExitErr{Code: ExitUsage, Err: fmt.Errorf(format, args...)}
}

func newUsageError(err error) error {
	if err == nil {
		return nil
	}

	return &ExitErr{Code: ExitUsage, Err: err}
}

func newParser(description string) (*kong.Kong, *CLI, error) {
	envMode := outfmt.FromEnv()
	vars := kong.Vars{
		"color":            envOr("NUBE_COLOR", colorAuto),
		"enabled_commands": envOr("NUBE_ENABLE_COMMANDS", ""),
		"json":             boolString(envMode.JSON),
		"plain":            boolString(envMode.Plain),
		"version":          VersionString(),
	}

	cli := &CLI{}
	parser, err := kong.New(
		cli,
		kong.Name("nube"),
		kong.Description(description),
		kong.Vars(vars),
		kong.Writers(os.Stdout, os.Stderr),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
	)
	if err != nil {
		return nil, nil, err
	}

	return parser, cli, nil
}

func baseDescription() string {
	return "Tienda Nube CLI for managing stores, products, orders, and more"
}

func helpDescription() string {
	desc := baseDescription()

	credPath, err := credstore.Path()
	credLine := "unknown"

	if err != nil {
		credLine = fmt.Sprintf("error: %v", err)
	} else if credPath != "" {
		credLine = credPath
	}

	return fmt.Sprintf("%s\n\nCredentials: %s", desc, credLine)
}
