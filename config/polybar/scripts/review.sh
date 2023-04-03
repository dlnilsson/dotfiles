#!/usr/bin/env bash

set -eou pipefail


GITHUB_NOTIFICATION="https://github.com/pulls/review-requested"

main() {
    count=$(gh search prs --draft=false --review-requested=@me  --state=open --json url | jq '. | length')
    if [[ $count -gt 0 ]]; then
        echo -e "\uf12a Review requested: $count "
    fi
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
