// Package reviews defines the git-insights reviews Cobra command.
package reviews

import (
	"github.com/dlnilsson/dotfiles/git-insights/cmd/countarg"
	"github.com/dlnilsson/dotfiles/git-insights/internal/leaderboard"
	"github.com/spf13/cobra"
)

// New creates the reviews command.
func New() *cobra.Command {
	opts := leaderboard.Options{Reviews: true}
	cmd := &cobra.Command{
		Use:   "reviews [count]",
		Short: "Rank who reviewed the most recently merged MRs",
		Long: `Leaderboard of who reviewed (approved) the last count merged MRs
(default 200), skipping MRs labeled Release. Self-approval by the author is not
counted.`,
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
	cmd.Flags().BoolVar(&opts.Timeline, "timeline", false,
		"print one CSV row per review (Merged,MR,Author,Reviewer), oldest first")
	cmd.MarkFlagsMutuallyExclusive("csv", "timeline")
	return cmd
}
