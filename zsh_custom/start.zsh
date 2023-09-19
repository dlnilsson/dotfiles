# autoload -U +X compinit && compinit
# autoload -U +X bashcompinit && bashcompinit

autoload -Uz compinit
compinit


fpath=($HOME/completions $fpath)

if [ -d "$HOME/completions" ]; then
	for f in $HOME/completions/*; do source $f; done
fi

if [ -d "$HOME/lib/azure-cli" ]; then
    source $HOME/lib/azure-cli/az.completion
fi
