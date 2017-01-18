set -g -x PATH /usr/local/bin $PATH
set -g -x PATH /Users/dln/.composer/vendor/bin $PATH
set -g -x PATH /Users/dln/.node/bin $PATH
set -g -x PATH /Users/dln/.nvm $PATH
set -g -x PATH /usr/local/Cellar/bison/3.0.4/bin $PATH
set -g -x PATH /usr/local/opt/go/libexec/bin $PATH
set -x NVM_DIR /Users/dln/.nvm

# go path
set -x GOPATH $HOME/go
set -g -x PATH $GOPATH/bin $PATH

##  dotfiles
set -gx PATH "$HOME/.dotfiles/bin" $PATH




# Path to Oh My Fish install.
set -q XDG_DATA_HOME
  and set -gx OMF_PATH "$XDG_DATA_HOME/omf"
  or set -gx OMF_PATH "$HOME/.local/share/omf"

# Customize Oh My Fish configuration path.
#set -gx OMF_CONFIG "/Users/dln/.config/omf"

# Load oh-my-fish configuration.
source $OMF_PATH/init.fish


alias cgs "clear; git status"
alias lla "ls -la"
alias pa "php artisan"
alias devup "cd ~/box/development; vagrant up"
alias puf "phpunit --verbose --debug --filter="
alias g "git"
alias nah="git reset --hard; git clean -df"

test -e {$HOME}/.iterm2_shell_integration.fish ; and source {$HOME}/.iterm2_shell_integration.fish


bash {$HOME}/.config/fish/init-gpg.sh
