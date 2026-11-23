// Package execute runs the git-insights Cobra tree and reports its error.
package execute

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// Execute runs root and prints a returned error to its stderr, prefixed with
// the command name. Callers only choose the exit status from the error.
func Execute(ctx context.Context, root *cobra.Command) error {
	err := root.ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintf(root.ErrOrStderr(), "%s: %v\n", root.Name(), err)
	}
	return err
}
