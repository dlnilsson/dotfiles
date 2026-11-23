// Package approvals defines the git-insights approvals Cobra command.
package approvals

import (
	"github.com/dlnilsson/dotfiles/git-insights/cmd/countarg"
	"github.com/dlnilsson/dotfiles/git-insights/internal/leaderboard"
	"github.com/spf13/cobra"
)

// New creates the approvals command.
func New() *cobra.Command {
	var opts leaderboard.Options
	cmd := &cobra.Command{
		Use:   "approvals [count]",
		Short: "Rank who approved the most recently merged MRs",
		Long: `Leaderboard of who approved the last count merged MRs (default 200),
skipping MRs labeled Release. Self-approval by the author counts.`,
		Args: countarg.Args,
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := countarg.Parse(args, leaderboard.DefaultCount)
			if err != nil {
				return err
			}
			opts.Count = count
			return leaderboard.Run(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.CSV, "csv", false, "print the leaderboard as CSV")
	return cmd
}
