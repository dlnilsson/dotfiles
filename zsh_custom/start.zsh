
fpath=($HOME/completions $fpath)

autoload -U +X compinit && compinit
autoload -U +X bashcompinit && bashcompinit

source $HOME/completions/*.bash
