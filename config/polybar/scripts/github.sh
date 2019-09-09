#!/usr/bin/env bash

set -eou pipefail

source $HOME/.github
CACHE=$HOME/.cache/github-notifications
ENTERPRISE_NOTIFICATION=${ENTERPRISE_URL/api\/v3/notifications}

teardown() {
	echo " Github down."
}
trap teardown ERR
main() {
	gh=$(curl -fs https://api.github.com/notifications?access_token=$GITHUB_NOTIFICATION_TOKEN 2>/dev/null)
	count=$(echo $gh | jq -r ". | length")
	output=""
	if [[ $count -gt 0 ]]; then
		output+=" $count \t"
	fi

	if [[ ! -z $ENTERPRISE_URL ]]; then
		gh=$(curl -fs --max-time 3 $ENTERPRISE_URL/notifications\?access_token\=$ENTERPRISE_NOTIFICATION_TOKEN 2>/dev/null)
		count=$(echo $gh | jq -r ". | length")
		for row in $(echo "${gh}" | jq -r '.[] | @base64'); do
			_jq() {
				echo ${row} | base64 --decode | jq -r ${1}
			}
			_id=$(_jq '.id')
			_title=$(_jq $ '.subject.title')
			_url=$(_jq $ '.subject.url') # api url, not HTML
			if [[ ! $(grep "$_id" "$CACHE") ]]; then
				echo $_title
				notify-send.sh " Github" "$_title" \
				-a "github" \
				-o "View:xdg-open $ENTERPRISE_NOTIFICATION &>/dev/null"
				echo $_id >> $CACHE
				sleep 1
			fi
		done
		if [[ $count -gt 0 ]]; then
			output+="\t\t   work: $count"
		fi
	fi
	echo -e "$output"
}

case "${1:-""}" in
	left_click)
		echo "left"
		sleep .1
		xdg-open "https://github.com/notifications"
		exit 0
	;;
	right_click)
		echo "right"
		sleep .1
		xdg-open $(echo $ENTERPRISE_NOTIFICATION)
		exit 0
	;;
	*)
		main
esac
