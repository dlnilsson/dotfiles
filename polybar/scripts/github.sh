#!/usr/bin/env bash

set -eou pipefail


source $HOME/.github

main() {
	gh=$(curl -fs https://api.github.com/notifications?access_token=$GITHUB_NOTIFICATION_TOKEN)
	count=$(echo $gh | jq -r ". | length")
	output=""
	if [[ $count -gt 0 ]]; then
		output+=" $count \t"
	fi

	if [[ ! -z $ENTERPRISE_URL ]]; then
		gh=$(curl -fs $ENTERPRISE_URL/notifications\?access_token\=$ENTERPRISE_NOTIFICATION_TOKEN)
		count=$(echo $gh | jq -r ". | length")

		if [[ $count -gt 0 ]]; then
			output+="\t\t  work: $count"
		fi
	fi
	echo -e $output
}

open_url() {
	case $( uname -s ) in
			Darwin)   open='open';;
			MINGW*)   open='start';;
			MSYS*)    open='start';;
			CYGWIN*)  open='cygstart';;
		*)
		if uname -r | grep -q Microsoft; then
			open='powershell.exe Start'
		else
			open='xdg-open'
		fi;;
	esac

	if [[ $BROWSER != "echo" ]]; then
		exec &>/dev/null
	fi

	${BROWSER:-$open} "$1"
}

case "${1:-""}" in
	left_click)
		open_url "https://github.com/notifications"
		exit 0
	;;
	right_click)
		open_url $(echo ${ENTERPRISE_URL/api\/v3/notifications})
		exit 0
	;;
	*)
		main $@
esac
