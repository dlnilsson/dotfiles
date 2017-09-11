
export GOPATH=$HOME/go
export PATH=/Users/dln/.composer/vendor/bin:$PATH
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/dotfiles/bin:$PATH
export PATH=/usr/local/Cellar/bison/3.0.4/bin:$PATH
export PATH=/Users/dln/.node/bin:$PATH
export PATH=/usr/local/sbin:$PATH


export EDITOR=nano

alias cgs="clear; git status"
alias lla="ls -la"
alias pa="php artisan"
alias puf="phpunit --verbose --debug --filter="
alias g="git"
alias nah="git reset --hard; git clean -df"
alias compsoer="composer"


#plugins=(git z zsh-autosuggestions)

drc() {
	echo "\nRemoving docker containers\n";
	for i in $(docker ps -aq); do docker rm -f $i; done
}
dri() {
	echo "\nRemoving docker images\n";
	for i in $(docker images -q); do docker rmi $i; done
}

inteleon() {
	ssh -i ~/.ssh/amazon.pem ubuntu@$*
}
aws_login() {
	eval ${$(aws ecr get-login --no-include-email)}
}
flushdns() {
	sudo dscacheutil -flushcache;sudo killall -HUP mDNSResponder
	local green=$(tput setaf 2)
	local reset=$(tput sgr0)
	echo -e "${green}\n dns flushed ${reset}"
}
composer_install() {
	docker run --rm -ti \
	-v $(pwd):/tmp \
	-v $HOME/.composer/cache:/root/.composer/cache \
	***REMOVED***/php-utilities \
	composer install --working-dir=/tmp --ignore-platform-reqs --no-scripts --no-suggest
}
code() {
	if [[ $# = 0 ]]
	then
		open -a "Visual Studio Code" -n
	else
		[[ $1 = /* ]] && F="$1" || F="$PWD/${1#./}"
		open -a "Visual Studio Code" -n --args "$F"
	fi
}
