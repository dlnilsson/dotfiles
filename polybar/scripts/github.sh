#!/usr/bin/env bash

source $HOME/.github


gh=$(curl -fs https://api.github.com/notifications?access_token=$GITHUB_NOTIFICATION_TOKEN)
count=$(echo $gh | jq -r ". | length")
output=""
if [[ $count -gt 0 ]]; then
	output+=" $count \t"
fi

if [[ ! -z $ENTERPRISE_URL ]]; then
	gh=$(curl -fs $ENTERPRISE_URL/notifications?access_token=$ENTERPRISE_NOTIFICATION_TOKEN)
	count=$(echo $gh | jq -r ". | length")

	if [[ $count -gt 0 ]]; then
		output+="\t\t  work: $count"
	fi
fi

echo -e $output
