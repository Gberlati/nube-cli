package cmd

import (
	"github.com/alecthomas/kong"
)

type HelpCmd struct{}

func (c *HelpCmd) Run(app *kong.Kong) error {
	_, err := app.Parse([]string{"--help"})
	return err
}
