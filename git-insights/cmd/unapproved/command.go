// Package unapproved defines the git-insights unapproved Cobra command.
package unapproved

import (
	"github.com/dlnilsson/dotfiles/git-insights/cmd/countarg"
	appunapproved "github.com/dlnilsson/dotfiles/git-insights/internal/unapproved"
	"github.com/spf13/cobra"
)

// New creates the unapproved command.
func New() *cobra.Command {
	var opts appunapproved.Options
	cmd := &cobra.Command{
		Use:   "unapproved [count]",
		Short: "List recently merged MRs that nobody approved",
		Long: `List the last count merged MRs that nobody approved (default 50), skipping
MRs labeled Release. Self-approval by the author counts as unapproved.`,
		Args: countarg.Args,
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := countarg.Parse(args, appunapproved.DefaultCount)
			if err != nil {
				return err
			}
			opts.Count = count
			return appunapproved.Run(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.CSV, "csv", false, "print the MR list as CSV")
	return cmd
}
