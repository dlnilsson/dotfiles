#!/usr/bin/env bash

set -eou pipefail


GITHUB_NOTIFICATION="https://github.com/pulls/review-requested"

main() {
    output=""
#    count=$(gh search prs --draft=false --review-requested=@me  --state=open --json url | jq '. | length' 2> /dev/null)
#    if [[ $count -gt 0 ]]; then
#        output+="\uf12a $count "
#    fi
    echo -e $output
}


case "${1:-""}" in
	left_click)
		echo "left"
		sleep .1
		xdg-open "$GITHUB_NOTIFICATION" &> /dev/null
		exit 0
	;;
	*)
		main
esac
