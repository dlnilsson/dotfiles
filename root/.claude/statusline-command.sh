#!/bin/bash
# Claude Code status line - materialshell theme
input=$(cat)
# printf '%s' "$input" >/tmp/statusline-input-latest

cwd=$(echo "$input" | jq -r '.workspace.current_dir')
model=$(echo "$input" | jq -r '.model.display_name')
context=$(echo "$input" | jq -r '.context_window.remaining_percentage // empty')
cost=$(echo "$input" | jq -r '.cost.total_cost_usd // empty')

session_id=$(echo "$input" | jq -r '.session_id')
agentrc_file="$cwd/.agentrc"


kitty @ --to "$KITTY_LISTEN_ON" set-window-title "Claude $cwd using $model"

touch "$agentrc_file"
if grep -q '^export CLAUDE_SESSION_ID=' "$agentrc_file"; then
    sed -i 's/^export CLAUDE_SESSION_ID=.*/export CLAUDE_SESSION_ID='"$session_id"'/' "$agentrc_file"
else
    echo "export CLAUDE_SESSION_ID=$session_id" >> "$agentrc_file"
fi

# Shorten home dir to ~
cwd="${cwd/#$HOME/\~}"

# Green dir, white "on", blue branch, dirty/clean indicator
printf '\033[32m%s\033[0m ' "$cwd"

if git -C "${cwd/#\~/$HOME}" rev-parse --git-dir >/dev/null 2>&1; then
    branch=$(git -C "${cwd/#\~/$HOME}" branch --show-current 2>/dev/null)
    if [ -n "$branch" ]; then
        if git -C "${cwd/#\~/$HOME}" diff-index --quiet HEAD -- 2>/dev/null; then
            indicator='\033[32m✔\033[0m'
        else
            indicator='\033[31m✗\033[0m'
        fi
        printf '\033[37mon \033[34m%s\033[0m %b ' "$branch" "$indicator"
        kitty @ --to "$KITTY_LISTEN_ON" set-window-title "Claude $cwd on $branch $model"
    fi
fi

printf '\033[37m| \033[34m%s\033[0m' "$model"

if [ -n "$context" ]; then
    printf ' \033[37m| \033[33m%s%%\033[0m' "$context"
fi

if [ -n "$cost" ]; then
    cost_fmt=$(printf '$%.3f USD' "$cost")
    printf ' \033[37m| \033[32m%s\033[0m' "$cost_fmt"
fi
