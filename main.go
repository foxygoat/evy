//go:build !tinygo

package main

import (
	"fmt"
	"io"
	"os"

	"foxygo.at/evy/pkg/evaluator"
	"foxygo.at/evy/pkg/lexer"
	"foxygo.at/evy/pkg/parser"
	"github.com/alecthomas/kong"
)

var version string = "v0.0.0"

const description = `
evy is a tool for managing evy source code.
`

type config struct {
	Version  kong.VersionFlag `short:"V" help:"Print version information"`
	Run      cmdRun           `cmd:"" help:"Run evy program"`
	Tokenize cmdTokenize      `cmd:"" help:"Tokenize evy program"`
	Parse    cmdParse         `cmd:"" help:"Parse evy program"`
}

type cmdRun struct {
	Source string `arg:"" help:"Source file. Default stdin" default:"-"`
}

type cmdTokenize struct {
	Source string `arg:"" help:"Source file. Default stdin" default:"-"`
}

type cmdParse struct {
	Source string `arg:"" help:"Source file. Default stdin" default:"-"`
}

func (c *cmdRun) Run() error {
	b, err := fileBytes(c.Source)
	if err != nil {
		return err
	}
	printFunc := func(s string) { fmt.Print(s) }
	evaluator.Run(string(b), printFunc)
	return nil
}

func (c *cmdTokenize) Run() error {
	b, err := fileBytes(c.Source)
	if err != nil {
		return err
	}
	result := lexer.Run(string(b))
	fmt.Println(result)
	return nil
}

func (c *cmdParse) Run() error {
	b, err := fileBytes(c.Source)
	if err != nil {
		return err
	}
	printFunc := func(s string) { fmt.Print(s) }
	rt := evaluator.Runtime{Print: printFunc}
	builtins := evaluator.DefaultBuiltins(rt).Decls()
	result := parser.Run(string(b), builtins)
	fmt.Println(result)
	return nil
}

func main() {
	kctx := kong.Parse(&config{},
		kong.Description(description),
		kong.Vars{"version": version},
	)
	kctx.FatalIfErrorf(kctx.Run())
}

func fileBytes(filename string) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filename)
}
