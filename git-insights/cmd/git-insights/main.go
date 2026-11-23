// Command git-insights shows git history insights and merge request reports.
package main

import (
	"context"
	"os"

	"github.com/dlnilsson/dotfiles/git-insights/cmd/execute"
	"github.com/dlnilsson/dotfiles/git-insights/cmd/root"
)

func main() {
	if err := execute.Execute(context.Background(), root.New()); err != nil {
		os.Exit(1)
	}
}
