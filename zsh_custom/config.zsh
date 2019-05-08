export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/dotfiles/bin:$PATH
#export PATH=$HOME/Library/Python/3.6/bin:$PATH
#export PATH=$HOME/Library/Python/3.7/bin:$PATH
#export PATH=/usr/local/sbin:$PATH
export PATH=$HOME/.local/bin:$PATH
#export COMPOSER_HOME=$HOME/.composer
#export PATH=$COMPOSER_HOME/vendor/bin:$PATH
#export COMPOSER_CACHE_DIR=$COMPOSER_HOME/cache
# export FLUTTER=$HOME/dev/flutter
# export PATH=$FLUTTER/bin:$PATH
# export PYENV_ROOT="$HOME/.pyenv"
# export PATH="$PYENV_ROOT/bin:$PATH"
export EDITOR=nano
export LANG=en_US.UTF-8


alias cgs="clear; git status"
alias lla="ls -la"
alias lal=lla
alias pa="php artisan"
alias puf="phpunit --verbose --debug --filter="
alias g="git"
alias nah="git reset --hard; git clean -df"
alias compsoer="composer"
#alias awk="gawk"

alias pbcopy='xsel --clipboard --input'
alias pbpaste='xsel --clipboard --output'


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

alias dps='docker ps -a'
alias dia="docker images -a"
alias dsl='docker service ls'
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
	local resp="$(curl -s ifconfig.co/json 2> /dev/null)"
	if [[ $resp ]]; then
		local ip=$(echo $resp | jq '.ip' -r)
		echo -n $ip | pbcopy
		info_msg $ip
	else
		warning_msg "Request failed. 🙀"
	fi
}
dbash() {
	if [[ $# = 0  ]]; then
		echo "No container id given"; exit 1;
	fi
	docker exec -it $1 bash
}
dsh() {
	if [[ $# = 0  ]]; then
		echo "No container id given"; exit 1;
	fi
	docker exec -it $1 sh
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
vacuum() {
	find . -name '*.zip' -o -name '.terraform' -o -name "*.tfstate.backup" -o -name ".DS_Store" | xargs rm -rf -
}
ff() {
	find . -type f -name $1
}

# https://junegunn.kr/2015/04/browsing-chrome-history-with-fzf/
chistory() {
	local cols sep
	cols=$(( COLUMNS / 3 ))
	sep='{::}'

	cp -f ~/Library/Application\ Support/Google/Chrome/Default/History /tmp/h

	sqlite3 -separator $sep /tmp/h \
	"select substr(title, 1, $cols), url
	from urls order by last_visit_time desc" |
	awk -F $sep '{printf "%-'$cols's  \x1b[36m%s\x1b[m\n", $1, $2}' |
	fzf --ansi --multi | sed 's#.*\(https*://\)#\1#' | xargs open
}
goland() {
	/usr/local/bin/goland .
}
#source <(awless completion zsh)
source "/usr/share/fzf/key-bindings.zsh"
source "/usr/share/fzf/completion.zsh" 2> /dev/null
source "/usr/share/zsh/site-functions"
# eval "$(pipenv --completion)"
