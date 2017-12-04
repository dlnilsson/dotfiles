#!/bin/bash
# Fork of https://github.com/paulirish/dotfiles/blob/master/brew.sh
#
# Install command-line tools using Homebrew
#

# Make sure we’re using the latest Homebrew
brew update

# Upgrade any already-installed formulae
brew upgrade


# GNU core utilities (those that come with OS X are outdated)
brew install coreutils
brew install moreutils
# GNU `find`, `locate`, `updatedb`, and `xargs`, `g`-prefixed
brew install findutils
# GNU `sed`, overwriting the built-in `sed`
brew install gnu-sed --default-names

brew cask install jumpcut

# Fonts
brew tap caskroom/fonts
brew cask install font-fira-code

# Bash 4
# Note: don’t forget to add `/usr/local/bin/bash` to `/etc/shells` before running `chsh`.
brew install bash
brew tap homebrew/versions
brew install bash-completion2

brew install homebrew/completions/brew-cask-completion

# generic colouriser  http://kassiopeia.juls.savba.sk/~garabik/software/grc/
brew install grc

# JAVA
brew install gradle

# Install wget with IRI support
brew install wget --with-iri

# Install more recent versions of some OS X tools
brew install vim --override-system-vi
brew install homebrew/dupes/grep
brew install homebrew/dupes/openssh
brew install homebrew/dupes/screen


# run this script when this file changes guy.
brew install entr

# github util. imho better than hub
brew install gh


# mtr - ping & traceroute. best.
brew install mtr

# allow mtr to run without sudo
mtrlocation=$(brew info mtr | grep Cellar | sed -e 's/ (.*//') #  e.g. `/Users/paulirish/.homebrew/Cellar/mtr/0.86`
sudo chmod 4755 $mtrlocation/sbin/mtr
sudo chown root $mtrlocation/sbin/mtr


## Upgrade GNU MAKE
brew info homebrew/dupes/make --with-default-names


# Install other useful binaries
brew install the_silver_searcher
brew install fzf

brew install git
brew tap git-duet/tap
brew install git-duet

brew install imagemagick --with-webp
brew install node # This installs `npm` too using the recommended installation method
brew install pv
brew install rename
brew install tree
brew install zopfli
brew install ffmpeg --with-libvpx

brew install terminal-notifier

brew install pidcat   # colored logcat guy

brew install zsh

# php
brew tap homebrew/dupes
brew tap homebrew/php
brew install homebrew/php/php70
brew install php70-intl
brew install composer

# go
brew install go --cross-compile-common
brew install glide

brew install cmake

# install the python package provided with homebrew
brew install python
brew install pip

brew install git-lfs

## Amazon cli
brew install jq
brew install jsonpp
brew install awscli

brew tap wallix/awless
brew install awless
brew install vault

# Terraform
brew install terraform

## GPG
brew install gpg
brew install keybase
brew install gpg-agent
brew install pinentry-mac

#
brew install graphviz
brew install cloc
brew install nethogs



# Remove outdated versions from the cellar
brew cleanup
