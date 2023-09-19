#!/usr/bin/env bash

set -eou pipefail

source "$HOME/.github"
CACHE=$HOME/.cache/github-notifications
NID=$HOME/.cache/github-nid
GITHUB_NOTIFICATION="https://github.com/notifications?query=is:unread"

teardown() {
	echo " Github down."
}
trap teardown ERR
main() {
	if [[ ! -v GITHUB_NOTIFICATION_TOKEN ]]; then
		exit 0
	fi
	gh=$(curl -fs -H "Authorization: token $GITHUB_NOTIFICATION_TOKEN" https://api.github.com/notifications 2>/dev/null)
	count=$(echo $gh | jq -r ". | length")
	output=""
    for row in $(echo "${gh}" | jq -r '.[] | @base64'); do
        _jq() {
            echo ${row} | base64 --decode | jq -r ${1}
        }
        _id=$(_jq '.id')
        _title=$(_jq $ '.subject.title')
        _url=$(_jq $ '.subject.url') # api url, not HTML
        if [[ ! $(grep "$_id" "$CACHE") ]]; then
            nid=$(cat $NID)

            echo "$_id" >> $CACHE
            notify-send " Github" "$_title" \
            -a "github"
            sleep 3
        fi
        sleep .5
    done
	if [[ $count -gt 0 ]]; then
		output+=" $count \t"
	fi
	echo -e "$output"
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
