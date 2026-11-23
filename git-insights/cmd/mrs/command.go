// Package mrs defines the git-insights mrs Cobra command.
package mrs

import (
	appmrs "github.com/dlnilsson/dotfiles/git-insights/internal/mrs"
	"github.com/spf13/cobra"
)

// New creates the mrs command and the hidden commands fzf calls back into.
func New() *cobra.Command {
	var opts appmrs.Options
	cmd := &cobra.Command{
		Use:   "mrs [team]",
		Short: "List open merge requests, optionally for one team",
		Long: `List open merge requests, optionally only those authored by members of team.

With --dmenu it picks with a noctalia dmenu, on a terminal with fzf it picks
with fzf, otherwise it prints TSV. Teams are read from $GIT_INSIGHTS_TEAMS_FILE,
else $XDG_CONFIG_HOME/git-insights/teams.json.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Team = args[0]
			}
			return appmrs.Run(cmd.OutOrStdout(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Dmenu, "dmenu", false, "pick with a noctalia dmenu")
	cmd.AddCommand(
		&cobra.Command{
			Use:    "open-url url",
			Hidden: true,
			Args:   cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return appmrs.OpenURL(args[0])
			},
		},
		&cobra.Command{
			Use:    "list-rows [team]",
			Hidden: true,
			Args:   cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				team := ""
				if len(args) == 1 {
					team = args[0]
				}
				return appmrs.ListRows(cmd.OutOrStdout(), team)
			},
		},
		&cobra.Command{
			Use:    "describe url",
			Hidden: true,
			Args:   cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return appmrs.Describe(cmd.OutOrStdout(), args[0])
			},
		},
	)
	return cmd
}
