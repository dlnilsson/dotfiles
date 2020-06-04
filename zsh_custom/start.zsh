autoload -U +X compinit && compinit
autoload -U +X bashcompinit && bashcompinit
fpath=($HOME/completions $fpath)

if [ -d "$HOME/completions" ]; then

    source $HOME/completions/*.bash
fi

if [ -d "$HOME/lib/azure-cli" ]; then
    source $HOME/lib/azure-cli/az.completion
fi

# if [ -d "$HOME/.cache/yay/azure-cli/src/azure-cli" ]; then
# 	source "$HOME/.cache/yay/azure-cli/src/azure-cli/az.completion"
# fi


