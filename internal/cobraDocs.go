package main

import (
	"github.com/thzinc/gunviolencearchive-client/package/cmd/cli"

	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTree(cli.RootCmd, "docs/gva")
	if err != nil {
		panic(err)
	}
}
