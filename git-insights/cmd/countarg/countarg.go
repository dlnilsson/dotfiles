// Package countarg parses the optional [count] positional argument shared by
// the merge request reporting commands.
package countarg

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// Args accepts zero or one positional argument.
var Args = cobra.MaximumNArgs(1)

// Parse returns the positive count in args, or def when args is empty.
func Parse(args []string, def int) (int, error) {
	if len(args) == 0 {
		return def, nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("count must be a positive integer, got %q", args[0])
	}
	return n, nil
}
