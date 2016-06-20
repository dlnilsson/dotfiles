set -g -x PATH /usr/local/bin $PATH
set -g -x PATH /Users/dln/.composer/vendor/bin $PATH
set -g -x PATH /Users/dln/.node/bin $PATH
set -g -x PATH /Users/dln/.nvm $PATH

set -x ANDROID_HOME /usr/local/opt/android-sdk
set -x NVM_DIR /Users/dln/.nvm

bash $NVM_DIR/nvm.sh
# [ -s "$NVM_DIR/nvm.sh" ]; . "$NVM_DIR/nvm.sh"  # This loads nvm

alias cgs "clear; git status"
alias lla "ls -la"
alias pa "php artisan"
alias devup "cd ~/box/development; vagrant up"

