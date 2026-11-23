// Package root assembles the git-insights command tree.
package root

import (
	"os"

	"github.com/dlnilsson/dotfiles/git-insights/cmd/approvals"
	"github.com/dlnilsson/dotfiles/git-insights/cmd/mrs"
	"github.com/dlnilsson/dotfiles/git-insights/cmd/reviews"
	"github.com/dlnilsson/dotfiles/git-insights/cmd/unapproved"
	"github.com/dlnilsson/dotfiles/git-insights/internal/insights"
	"github.com/spf13/cobra"
)

// New creates the complete git-insights command tree. Without a subcommand it
// opens the interactive TUI.
func New() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:           "git-insights",
		Short:         "Git history insights and merge request reports",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		Long: `git-insights shows git history insights for the repository in the current
directory. Without a subcommand it opens an interactive TUI.

The merge request commands talk to the forge of the origin remote through its
CLI (glab for GitLab).`,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if dir == "" {
				return nil
			}
			return os.Chdir(dir)
		},
		RunE: func(*cobra.Command, []string) error {
			return insights.Run()
		},
	}
	cmd.PersistentFlags().StringVarP(&dir, "directory", "C", "", "run as if started in `dir`")
	cmd.AddCommand(
		mrs.New(),
		approvals.New(),
		reviews.New(),
		unapproved.New(),
	)
	return cmd
}
