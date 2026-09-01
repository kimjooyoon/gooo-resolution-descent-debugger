package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-resolution-descent-debugger/internal/debugger"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(64)
	}
	var err error
	switch os.Args[1] {
	case "manifest":
		err = manifest(os.Args[2:])
	case "debug":
		err = debug(os.Args[2:])
	case "conformance":
		err = conformance(os.Args[2:])
	default:
		usage()
		os.Exit(64)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func manifest(args []string) error {
	flags := flag.NewFlagSet("manifest", flag.ContinueOnError)
	sourcePath := flags.String("source", ".gooo/resolution-descent-debugger.gooo", "authoritative .gooo source")
	contractPath := flags.String("contract", "contracts/denominator-v1.json", "fixed denominator contract")
	if err := flags.Parse(args); err != nil {
		return err
	}
	source, sourceRaw, err := debugger.LoadSource(*sourcePath)
	if err != nil {
		return err
	}
	denominator, contractRaw, err := debugger.LoadDenominator(*contractPath)
	if err != nil {
		return err
	}
	if err := debugger.ValidateBindings(source, denominator); err != nil {
		return err
	}
	ir := debugger.Lower(source, *sourcePath, *contractPath, debugger.DigestBytes(sourceRaw), debugger.DigestBytes(contractRaw))
	return writeStdout(ir)
}

func debug(args []string) error {
	flags := flag.NewFlagSet("debug", flag.ContinueOnError)
	root := flags.String("root", ".", "input repository root")
	sourcePath := flags.String("source", ".gooo/resolution-descent-debugger.gooo", "authoritative .gooo source")
	contractPath := flags.String("contract", "contracts/denominator-v1.json", "fixed denominator contract")
	fixturePath := flags.String("fixture", "", "caller-owned fixture JSON")
	out := flags.String("out", "", "empty caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *fixturePath == "" || *out == "" {
		return fmt.Errorf("debug requires --fixture and --out")
	}
	report, err := debugger.ExecuteSingle(*root, *out, *sourcePath, *contractPath, *fixturePath)
	if err != nil {
		return err
	}
	return writeStdout(report)
}

func conformance(args []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	root := flags.String("root", ".", "input repository root")
	sourcePath := flags.String("source", ".gooo/resolution-descent-debugger.gooo", "authoritative .gooo source")
	contractPath := flags.String("contract", "contracts/denominator-v1.json", "fixed denominator contract")
	out := flags.String("out", "", "empty caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("conformance requires --out")
	}
	report, err := debugger.ExecuteConformance(*root, *out, *sourcePath, *contractPath)
	if err != nil {
		return err
	}
	return writeStdout(report)
}

func writeStdout(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gooo-resolution-descent-debugger manifest|debug|conformance [flags]")
}
