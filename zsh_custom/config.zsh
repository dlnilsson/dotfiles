export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin
export PATH=$GOPATH/bin:$PATH
export PATH=$HOME/.dotfiles/bin:$PATH
export PATH=$HOME/.local/bin:$PATH
export PATH=$HOME/.npm-global/bin:$PATH
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"
export EDITOR=nano
export GLAB_PAGER=delta
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
alias ks="kubectl --context minikube --namespace srenity"
alias gitadd="git add"
alias yat="bat --language=yaml -p"
alias pacmane="pacman"
alias cgs="clear; git status"
alias las="ls"
alias ls="lsd"
alias fd="fdfind"
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
	items=()
	for img in $(giphy search $1); do
		items+=("$img")
	done

	res=$(printf '%s\n' "${items[@]}" | jq -R . \
	| jq -s . \
	| jq -r '.[]' \
	| fzf --preview-window 'up,85%,border-bottom,+{2}+3/3,~3' \
	--bind 'ctrl-y:execute-silent(echo {})+abort' \
	--preview='kitty icat --clear --transfer-mode=memory \
	--stdin=no --place=${FZF_PREVIEW_COLUMNS}x${FZF_PREVIEW_LINES}@0x0 {}
	')
	echo "![]($res)"| xsel --clipboard --input
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
	ID=$(docker ps --format '{{json  .ID}} {{json .Image}} {{json .Names}}'  | fzf --bind "enter:execute(echo {1} | cut -d '\"' -f 2 )+abort")
	if [[ ! -z $ID ]]; then
		docker inspect $ID | jq .
	fi

}
docker_logs() {
	ID=$(docker ps --format '{{json  .ID}} {{json .Image}} {{json .Names}}'  | fzf --bind "enter:execute(echo {1} | cut -d '\"' -f 2 )+abort")
	if [[ ! -z $ID ]]; then
		docker logs $ID
	fi

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
	docker exec -it $ID /bin/bash
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
			kitty @ send-text "\n"
			return
		fi
		dir=$(dirname "$dir")
	done
	warning_msg "No .git directory found in any parent directory." >&2
	kitty @ send-text "\n"
}


zle -N root
# showkey -a
#bindkey '^[[H' rootdwqd
bindkey '^[[1;5H' root

jwt() {
	jq -R 'split(".") | .[1] | @base64d | fromjson' <<< "$1"
}

kslogs() {
	res=$(ks get pods | awk 'NR>1 { print $1}' | fzf)
	TERM_TITLE="$res" ks logs $res -f
}

rgd() {
	 rg --json -C 5 $1 | delta
}

ars() {
	answer=$(gum choose $(autorandr --list) --selected=$(autorandr --current))
	if [ ! $? -eq 0 ]; then
		return
	fi
	info_msg "loading in $answer"
	autorandr --load "$answer"
	feh --recursive --randomize --bg-fill ~/Wallpapers &> /dev/null
}


alias jqd="jq 'with_entries(.value |= @base64d)'"

qrcode() {
	echo $1 | qr
}
