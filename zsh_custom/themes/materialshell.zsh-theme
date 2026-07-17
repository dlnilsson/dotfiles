# https://github.com/carloscuesta/materialshell
eval red=$fg[red]
eval green=$fg[green]
eval yellow=$fg[yellow]
eval blue=$fg[blue]
eval magenta=$fg[magenta]
eval cyan=$fg[cyan]
eval white=$fg[white]
eval grey=$fg[grey]

ZSH_THEME_DOCKER_PROMPT_SHA_BEFORE="%{$white%}[%{$cyan%}"
ZSH_THEME_DOCKER_PROMPT_SHA_AFTER="%{$white%}]"

# PROMPT='$(_user_host)${_current_dir}$(git_prompt_info)
# PROMPT='${_current_dir}$(_wt_statusline)
PROMPT='${_current_dir}$(git_prompt_info)
%{$white%}>%{$reset_color%} '
PROMPT2='%{$grey%}◀%{$reset_color%} '
#RPROMPT='$(_vi_status)%{$(echotc UP 1)%}$(_docker_info)$(_kubectl_ctx)$(git_remote_status) $(git_prompt_short_sha) ${_return_status} %{$white%}%T%{$(echotc DO 1)%}%{$reset_color%}'
#RPROMPT='$(_vi_status)%{$(echotc UP 1)%}$(_docker_info)$(_kubectl_ctx)$(git_remote_status) ${_return_status} %{$white%}%T%{$(echotc DO 1)%}%{$reset_color%}'
RPROMPT='$(_vi_status)%{$(echotc UP 1)%}$(_docker_info)$(_kubectl_ctx)$(git_remote_status) ${_return_status} %{$white%}%{$(echotc DO 1)%}%{$reset_color%}'

local _current_dir="%{$green%}%0~%{$reset_color%} "
local _return_status="%{$red%}%(?..×)%{$reset_color%}"

function _user_host() {
  echo "%{$red%}%n%{$reset_color%} %{$white%}in "
}
_kubectl_ctx() {
  echo ""
  # VAL=$(kubectl config current-context 2> /dev/null)
  # if [[ ! -z $VAL ]]; then
  #     echo "$ZSH_THEME_DOCKER_PROMPT_SHA_BEFORE$VAL$ZSH_THEME_DOCKER_PROMPT_SHA_AFTER"
  # fi
}
_docker_info() {
    local VAL
    case ${DOCKER_HOST:-} in
        "tcp://192.168.99.100:2376"|"tcp://192.168.99.104:2376")
            VAL=Manager
            ;;
        "tcp://192.168.99.101:2376"|"tcp://192.168.99.105:2376")
            VAL=Worker
            ;;
        *)
            VAL=$DOCKER_HOST
            ;;
    esac
    if [ ! -z $SSH_CLIENT ]; then
      VAL="SSH: ${HOST}"
    fi
    if [[ ! -z $VAL ]]; then
        echo "$ZSH_THEME_DOCKER_PROMPT_SHA_BEFORE$VAL$ZSH_THEME_DOCKER_PROMPT_SHA_AFTER"
    fi
}

typeset -g _wt_prompt=""
typeset -g _wt_async_pid=0

function _wt_statusline() {
  printf '%s' "$_wt_prompt"
}

function _wt_async_start() {
  if (( _wt_async_pid != 0 )); then
    kill -9 "$_wt_async_pid" 2>/dev/null
    _wt_async_pid=0
  fi

  local tmpfile="${TMPDIR:-/tmp}/zsh-wt-prompt-$$"
  setopt local_options no_monitor
  {
    local json branch statusline extra result=""
    json=$(wt list --format=json 2>/dev/null)
    if [[ $? -eq 0 ]]; then
      branch=$(jq -r '.[] | select(.is_current) | .branch' <<< "$json")
      if [[ -n "$branch" ]]; then
        statusline=$(jq -r '.[] | select(.is_current) | .statusline' <<< "$json")
        extra="${statusline#"$branch"}"
        result=$(printf '\033[37mon \033[34m%s\033[0m%b ' "$branch" "$extra")
      fi
    fi
    print -n "$result" > "$tmpfile"
    kill -USR1 $$ 2>/dev/null
  } &!
  _wt_async_pid=$!
}

function TRAPUSR1() {
  local tmpfile="${TMPDIR:-/tmp}/zsh-wt-prompt-$$"
  if [[ -f "$tmpfile" ]]; then
    _wt_prompt=$(<"$tmpfile")
    rm -f "$tmpfile"
  fi
  _wt_async_pid=0
  zle && zle reset-prompt
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd _wt_async_start

function _vi_status() {
  if {echo $fpath | grep -q "plugins/vi-mode"}; then
    echo "$(vi_mode_prompt_info)"
  fi
}

if [[ $USER == "root" ]]; then
  CARETCOLOR="$red"
else
  CARETCOLOR="$white"
fi

MODE_INDICATOR="%{_bold$yellow%}❮%{$reset_color%}%{$yellow%}❮❮%{$reset_color%}"

ZSH_THEME_GIT_PROMPT_PREFIX="%{$white%}on %{$blue%}"
ZSH_THEME_GIT_PROMPT_SUFFIX="%{$reset_color%} "

ZSH_THEME_GIT_PROMPT_DIRTY=" %{$red%}✗%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_CLEAN=" %{$green%}✔%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_BEHIND_REMOTE="%{$red%}⬇%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_AHEAD_REMOTE="%{$green%}⬆%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_DIVERGED_REMOTE="%{$yellow%}⬌%{$reset_color%}"

# Format for git_prompt_long_sha() and git_prompt_short_sha()
ZSH_THEME_GIT_PROMPT_SHA_BEFORE="%{$reset_color%}[%{$yellow%}"
ZSH_THEME_GIT_PROMPT_SHA_AFTER="%{$reset_color%}]"

# LS colors, made with http://geoff.greer.fm/lscolors/
export LSCOLORS="exfxcxdxbxegedabagacad"
export LS_COLORS='di=34;40:ln=35;40:so=32;40:pi=33;40:ex=31;40:bd=34;46:cd=34;43:su=0;41:sg=0;46:tw=0;42:ow=0;43:'
#  GREP_COLOR='1;33'
export GREP_COLORS='mt=1;33'
