
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/dotfiles/bin:$PATH
export PATH=/usr/local/Cellar/bison/3.0.4/bin:$PATH
export PATH=$HOME/.node/bin:$PATH
export PATH=$HOME/Library/Python/3.6/bin:$PATH
export PATH=/usr/local/sbin:$PATH
export COMPOSER_HOME=$HOME/.composer
export PATH=$COMPOSER_HOME/vendor/bin:$PATH
export COMPOSER_CACHE_DIR=$COMPOSER_HOME/cache
export FLUTTER=$HOME/dev/flutter
export PATH=$FLUTTER/bin:$PATH
# export PYENV_ROOT="$HOME/.pyenv"
# export PATH="$PYENV_ROOT/bin:$PATH"
export EDITOR=nano

alias cgs="clear; git status"
alias lla="ls -la"
alias lal=lla
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
			if [[ ! $(docker ps > /dev/null 2>&1) ]]; then
				notice_msg "Starting docker..."
				open --hide --background -a Docker
			else
				warning_msg "seems like docker daemon is already running..."
			fi
			;;
		'Linux' )
			# noop
			;;
		*)
		warning_msg 'Cant start docker daemon'
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

aws_login() {
	eval ${$(aws ecr get-login --no-include-email)}
}
flushdns() {
	sudo dscacheutil -flushcache;sudo killall -HUP mDNSResponder
	info_msg "DNS Flushed!"
}

json_pretty() {
	# npm install -g underscore-cli
	underscore print --outfmt pretty
}
alias jsonp=json_pretty
alias jsonpretty=json_pretty
alias dps='docker ps -a'
alias dia="docker images -a"

docker_inspect() {
	docker inspect $* | jq .
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
	local ip=$(http ifconfig.co/json | jq '.ip' -r)
	echo -n $ip | pbcopy
	info_msg $ip
}
dbash() {
	if [[ $# = 0  ]]; then
		echo "No container id given"; exit 1;
	fi
	docker exec -it $1 bash
}
docker_inspect_first() {
	if [[ $(docker ps | wc -l) -lt 2 ]]; then
		warning_msg "Seems like no docker container is running.."; exit 1;
	fi
	docker inspect $(docker ps | awk 'FNR==2 {print $1}') | jq .
}
# first docker container bash
fdb() {
	if [[ $(docker ps | wc -l) -lt 2 ]]; then
		warning_msg "Seems like no docker container is running.."; exit 1;
	fi
	docker exec -it $(docker ps | awk 'FNR==2 {print $1}') bash
}

# https://twitter.com/elithrar/status/971314557372239872?s=12
get_new_mac() {
	sudo /System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport -z \
	&& sudo ifconfig en0 ether a0$(openssl rand -hex 5 | sed 's/\(..\)/:\1/g') \
	&& networksetup -detectnewhardware
}
rmd () {
	pandoc $1 | lynx -stdin
}
alin() {
	awless ls instances --sort uptime --filter name=$1
}
aid() {
	awless ls instances --filter id=$1
}
aidssh() {
	inteleon $(awless ls instances --filter id=$1 | awk 'FNR==3 {print $13}')
}
vacuum() {
	find . -name '*.zip' -o -name '.terraform' -o -name "*.tfstate.backup" -o -name ".DS_Store" | xargs rm -rf -
}
ff() {
	find . -type f -name $1
}
source <(awless completion zsh)
source "/usr/local/opt/fzf/shell/key-bindings.zsh"
source "/usr/local/opt/fzf/shell/completion.zsh" 2> /dev/null
