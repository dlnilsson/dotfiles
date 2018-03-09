
export GOPATH=$HOME/go
export PATH=/Users/dln/.composer/vendor/bin:$PATH
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/dotfiles/bin:$PATH
export PATH=/usr/local/Cellar/bison/3.0.4/bin:$PATH
export PATH=/Users/dln/.node/bin:$PATH
export PATH=/usr/local/sbin:$PATH
export COMPOSER_HOME=$HOME/.composer
export COMPOSER_CACHE_DIR=$COMPOSER_HOME/cache

export EDITOR=nano
export AWS_ACC=***REMOVED***

alias cgs="clear; git status"
alias lla="ls -la"
alias pa="php artisan"
alias puf="phpunit --verbose --debug --filter="
alias g="git"
alias nah="git reset --hard; git clean -df"
alias compsoer="composer"

# Init jenv
if which jenv > /dev/null; then eval "$(jenv init -)"; fi

#plugins=(git z zsh-autosuggestions docker emoji)

info_msg() {
	local green=$(tput setaf 2)
	local reset=$(tput sgr0)
	echo -e "${green}$@${reset}"
}
notice_msg() {
	local blue=$(tput setaf 4)
	local reset=$(tput sgr0)
	echo -e "${blue}$@${reset}"
}
important_msg() {
	local yellow=$(tput setaf 3)
	local reset=$(tput sgr0)
	echo -e "${yellow}$@${reset}"
}

warning_msg() {
	local red=$(tput setaf 1)
	local reset=$(tput sgr0)
	echo -e "${red}$@${reset}"
}

docker_up() {
	local name=$(uname -s)
	case $name in
		'Darwin' )
			# psuedo-code, does not work, do another check if docker daemon is running.
			if [[ $(ps aux | grep Docker | wc -l) > 2 ]]; then
				important_msg "seems like docker daemon is already running..."
			else
				notice_msg "Starting docker..."
				open --hide --background -a Docker
			fi
			;;
		'Linux' )
			# noop
			;;
		*)
		echo 'Cant start docker daemon'
	esac
}
drc() {
	notice_msg "\nRemoving docker containers - $emoji[whale] \n";
	for i in $(docker ps -aq); do docker rm -f $i; done
}
drv() {
	notice_msg "\nRemoving docker volumes - $emoji[whale] \n";
	for i in $(docker volume ls -q); do docker volume rm -f $i; done
}
dri() {
	notice_msg "\nRemoving docker images $emoji[whale] \n";
	for i in $(docker images -q); do docker rmi $i; done
}
drn() {
	notice_msg "\nRemove all docket networks $emoji[whale]\n";
	docker network prune --force
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
	echo -e \U+1F602
	info_msg "DNS Flushed!"
}
composer_install() {
	docker run --rm -ti \
	-v $(pwd):/tmp/source \
	--workdir '/tmp/source' \
	-v $COMPOSER_CACHE_DIR:/root/.composer/cache \
	$AWS_ACC/php-utilities:latest \
	bash -c "composer install --working-dir=/tmp/source --ignore-platform-reqs --no-suggest"
}
c0mposer() {
	docker run --rm -ti \
	-v $(pwd):/tmp/source \
	--workdir '/tmp/source' \
	-v $COMPOSER_CACHE_DIR:/tmp/.composer/cache \
	$AWS_ACC/php-utilities:latest \
	composer $*
}
php_security_checker() {
	docker run --rm -ti \
	-v $(pwd):/tmp \
	$AWS_ACC/php-utilities:latest \
	security-checker security:check /tmp/composer.lock
}
json_pretty() {
	# npm install -g underscore-cli
	underscore print --outfmt pretty
}
alias jsonp=json_pretty
alias jsonpretty=json_pretty
alias dps='docker ps -a'
alias dia="docker images -a"

dcfg() {
	docker inspect $* | json_pretty
}
docker_env() {
	docker inspect $* | jq '.[0].Config.Env'
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
enter_base() {
	docker exec -it $(docker ps --filter='name=base' | awk 'NR>1 {print $1; exit}') bash
}
myip() {
	local ip=$(http ifconfig.co/json | jq '.ip')
	echo $ip | sed 's/^.\(.*\).$/\1/' | pbcopy
	info_msg $ip
}
dbash() {
	if [[ $# = 0  ]]; then
		echo "No container id given"; exit 1;
	fi
	docker exec -it $1 bash
}

# https://twitter.com/elithrar/status/971314557372239872?s=12
get_new_mac() {
	sudo /System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport -z \
	&& sudo ifconfig en0 ether a0$(openssl rand -hex 5 | sed 's/\(..\)/:\1/g') \
	&& networksetup -detectnewhardware
}
source <(awless completion zsh)
