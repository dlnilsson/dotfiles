package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

const unapprovedHelp = `usage: git-insights unapproved [-C dir] [--csv] [count]

Lists the most recently merged MRs that nobody approved (default: last 50),
skipping MRs labeled Release. Self-approval by the author counts as unapproved.

  -C dir      run in dir instead of the current directory
  --csv       print the MR list as CSV
  -h, --help  print this help`

func runUnapproved(args []string) error {
	var (
		count  = 50
		useCSV bool
	)
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "-h", "--help":
			fmt.Println(unapprovedHelp)
			return nil
		case "-C":
			i++
			if i >= len(args) {
				return errors.New(unapprovedHelp)
			}
			if err := os.Chdir(args[i]); err != nil {
				return err
			}
		case "--csv":
			useCSV = true
		default:
			n, err := strconv.Atoi(arg)
			if err != nil || n < 1 {
				return errors.New(unapprovedHelp)
			}
			count = n
		}
	}

	if !isGitRepo() {
		return errors.New("not a git repository")
	}
	forge, err := detectForge()
	if err != nil {
		return err
	}
	project, err := forge.Project()
	if err != nil {
		return err
	}

	mrs, _, err := collectMerged(forge, count,
		func(mr MergeRequest) bool { return !mr.Approved },
		func(found, scanned int) string {
			return fmt.Sprintf("Fetching merged MRs in %s (%d/%d unapproved, %d scanned)...", project, found, count, scanned)
		})
	if err != nil {
		return err
	}
	sortUnapproved(mrs)

	if useCSV {
		records := [][]string{{"MR", "URL", "Merged", "Author", "Title"}}
		for _, mr := range mrs {
			records = append(records, []string{strconv.Itoa(mr.Number), mr.URL, mergedDate(mr), mr.Author.Username, mr.Title})
		}
		return writeCSV(records)
	}

	rows := make([][]string, len(mrs))
	for i, mr := range mrs {
		ref := mr.Ref
		if isTerminal(os.Stdout) {
			// OSC 8 hyperlink, so the MR reference is clickable in the terminal.
			ref = "\033]8;;" + mr.URL + "\a" + ref + "\033]8;;\a"
		}
		rows[i] = []string{ref, mergedDate(mr), mr.Author.Username, mr.Title}
	}
	fmt.Println(renderTable([]string{"MR", "Merged", "Author", "Title"}, rows))

	perAuthor := rankAuthors(mrs)
	authorRows := make([][]string, len(perAuthor))
	for i, a := range perAuthor {
		authorRows[i] = []string{a.Username, strconv.Itoa(a.Count)}
	}
	fmt.Println(renderTable([]string{"Author", "Unapproved"}, authorRows))
	return nil
}

func mergedDate(mr MergeRequest) string {
	return mr.MergedAt.UTC().Format("2006-01-02")
}

// sortUnapproved sorts by author, then newest merge date first, then number.
func sortUnapproved(mrs []MergeRequest) {
	slices.SortFunc(mrs, func(a, b MergeRequest) int {
		return cmp.Or(
			strings.Compare(a.Author.Username, b.Author.Username),
			strings.Compare(mergedDate(b), mergedDate(a)),
			cmp.Compare(a.Number, b.Number),
		)
	})
}

// rankAuthors counts merge requests per author, most first, ties by name.
func rankAuthors(mrs []MergeRequest) []userCount {
	counts := make(map[string]int)
	for _, mr := range mrs {
		counts[mr.Author.Username]++
	}
	return rankCounts(counts)
}
