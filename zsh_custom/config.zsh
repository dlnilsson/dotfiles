export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/.dotfiles/bin:$PATH
export PATH=$HOME/.local/bin:$PATH
export PATH=$HOME/.npm-global/bin:$PATH
export EDITOR=nano
export GLAB_PAGER=cat
export LANG=en_US.UTF-8
#export BROWSER=google-chrome-stable
export BROWSER=firefox
export TERM=xterm-256color
export STEAM_FRAME_FORCE_CLOSE=1
export FZF_DEFAULT_COMMAND="ag -l -g ''"
export FZF_DEFAULT_OPTS=$FZF_DEFAULT_OPTS'
    --color=fg:#e5e9f0,bg:#3b4251,hl:#81a1c1
    --color=fg+:#eceff4,bg+:#4c566a,hl+:#8fbcbb
    --color=info:#eacb8a,prompt:#bf6069,pointer:#b48dac
    --color=marker:#a3be8b,spinner:#b48dac,header:#a3be8b
	--bind page-up:preview-up,page-down:preview-down'

# export TERMINAL=termite

alias yat="bat --language=yaml -p"
alias pacmane="pacman"
alias cgs="clear; git status"
alias las="ls"
alias ls="lsd"
alias tf="terraform"
alias bat="batcat"
alias cat="bat -pp"
alias cls="clear"
# alias tree="lsd -tree"
alias lla="ls -la"
alias lal=lla
alias pa="php artisan"
alias puf="phpunit --verbose --debug --filter="
alias g="git"
alias nah="git reset --hard; git clean -df"
alias compsoer="composer"
#alias awk="gawk"
alias pbpaste='xsel --clipboard --output'
alias dps='docker ps -a'
alias dia="docker images -a"
alias dsl='docker service ls'



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


# private enviornment variables
SECRET_ENV=$HOME/.secrets

if [[ ! -a $SECRET_ENV ]] then
	warning_msg $SECRET_ENV "not found."
else
	source $SECRET_ENV
fi

pbcopy() {
	if [ -z "$KITTY_WINDOW_ID" ] | [ -z "$DISPLAY" ]; then
		kitty +kitten clipboard
	else
		xsel --clipboard --input
	fi
}

open() {
	$(xdg-open "$@" &> /dev/null &)
}

giphymd() {
	info_msg "search giphy for $1"
	items=()
	for img in $(giphy search $1); do
		kitty +kitten icat --align left --silent $img
		items+=("![]($img)")
	done

	res=$(printf "%s\n" "${items[@]}" | fzf --layout=reverse --ansi --height 20% \
	--bind 'ctrl-y:execute-silent(echo {})+abort')
	echo $res | xsel --clipboard --input
	clear
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
docker_inspect() {
	docker inspect $* | jq .
}
docker_env() {
	docker inspect $* | jq '.[0].Config.Env'
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

dshell() {
	ID=$(docker ps --format "{{.ID}}\t{{.Names}}\t{{.Image }}" | fzf --height=40% --bind "enter:execute(echo {1})+abort")
	docker exec -it $ID /bin/zsh
	exit_code=$?
	if [ ! $exit_code -eq 0 ]; then
		important_msg "bash not found, using sh"
		docker exec -it $ID sh
	fi
}

dbash() {
	ID=$(docker ps --format "{{.ID}}\t{{.Names}}\t{{.Image }}" | fzf --height=40% --bind "enter:execute(echo {1})+abort")
	docker exec -it $ID bash
}
dsh() {
	ID=$(docker ps --format "{{.ID}}\t{{.Names}}\t{{.Image }}" | fzf --height=40% --bind "enter:execute(echo {1})+abort")
	docker exec -it $ID sh
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
	find . \
	-name '.terraform' \
	-or -name '*.tfstate.backup' \
	-or -name '.DS_Store' | xargs rm -rf -
	# find . -type f -name "*.py[co]" -delete -or -type d -name "__pycache__" -delete
}
ff() {
	find . -type f -name $1
}

# https://junegunn.kr/2015/04/browsing-chrome-history-with-fzf/
chistory() {
	local cols sep
	cols=$(( COLUMNS / 3 ))
	sep='{::}'
	local historyDB
	local openCMD
	case $( uname -s ) in
	  Darwin) historyDB=~/Library/Application\ Support/Google/Chrome/Default/History openCMD=open;;
	  *) historyDB=$HOME/.config/google-chrome/Default/History openCMD=xdg-open;;
	esac
	cp -f $historyDB /tmp/h

	sqlite3 -separator $sep /tmp/h \
	"select substr(title, 1, $cols), url
	from urls order by last_visit_time desc" |
	awk -F $sep '{printf "%-'$cols's  \x1b[36m%s\x1b[m\n", $1, $2}' |
	fzf --ansi --multi | sed 's#.*\(https*://\)#\1#' | xargs $openCMD 2> /dev/null
}
goland() {
	/usr/local/bin/goland .
}
kill_dnsmasq() {
	docker stop $(docker ps  --filter name=dnsmasq -qa)
	docker rm $(docker ps  --filter name=dnsmasq -qa)
}
run_dnsmasq() {
	docker run \
	--name dnsmasq \
	-d \
	-p 53:53/udp \
	-p 5380:8080 \
	-v $HOME/scripts/dnsmasq.conf:/etc/dnsmasq.conf \
	--log-opt "max-size=100m" \
	-e "HTTP_USER=foo" \
	-e "HTTP_PASS=bar" \
	--restart always \
	jpillora/dnsmasq
}
ping_sound() {
	paplay /usr/share/sounds/freedesktop/stereo/complete.oga &> /dev/null
}
imgcat() {
	if [ ! -z $KITTY_WINDOW_ID ]; then
		kitty +kitten icat $1
	else
		warning_msg "cant view image in this terminal."
		xdg-open $1
	fi
}
pretty_diff() {
		if [ ! -z $KITTY_WINDOW_ID ]; then
		kitty +kitten diff $1 $2
	else
		diff $1 $2
	fi
}
reload_gtk_theme() {
	theme=$(gsettings get org.gnome.desktop.interface gtk-theme)
	gsettings set org.gnome.desktop.interface gtk-theme ''
	sleep 1
	gsettings set org.gnome.desktop.interface gtk-theme $theme
}
# get human readable timestamp from unixtimestamp
uts() {
	date --date=\@$1
}
#TODO
#source <(awless completion zsh)
#source "/usr/share/fzf/key-bindings.zsh"
#source "/usr/share/fzf/completion.zsh" 2> /dev/null
#source "/usr/share/zsh/site-functions"
# eval "$(pipenv --completion)"


put_editorconfig() {
	if [[ $(git ls-files --error-unmatch .editorconfig 2> /dev/null) ]]; then
		echo 'file being tracked; do nothing'
	elif [[ -f .editorconfig ]]; then
		echo 'file exist already; do nothing'
	else
		echo 'creating symlink'
		ln -s $HOME/.dotfiles/.editorconfig $PWD
	fi
}


mr() {
glab mr list \
	| awk '{ if (NF > 0) if (NR>1) print substr($1,2);}' \
	| fzf --ansi \
	--color "hl:-1:underline,hl+:-1:underline:reverse" \
	--preview 'glab mr view {} --comments | batcat --plain --color=always && glab mr approvers {} | batcat --plain --color=always' \
	--preview-window 'up,85%,border-bottom,+{2}+3/3,~3' \
	--bind 'enter:become(glab mr view {} --web 2> /dev/null)'


}


kctx() {
	local current=$(kubectl config current-context &> /dev/null || echo "")
	answer=$(GUM_CHOOSE_SELECTED=$current gum choose --height 30 $(kubectl config get-contexts | awk '(NR>1) {print $2}'))
	if [[ ! -z $answer ]]; then
		kubectl config use-context $answer
	fi
}

gshow() {
	git log --oneline | \
	fzf --ansi --layout=reverse \
	--preview-window=right:65%:wrap \
	--preview 'git show {1} | delta --file-style=omit --width=${FZF_PREVIEW_COLUMNS:-$COLUMNS}' \
	--preview-window=up:90%:wrap
}

root() {
	local dir
	dir=$PWD

	while [ "$dir" != "/" ]; do
		if [ -d "$dir/.git" ]; then
			cd "$dir"
			return
		fi
		dir=$(dirname "$dir")
	done

	warning_msg "No .git directory found in any parent directory." >&2
}
