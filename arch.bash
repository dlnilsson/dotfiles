#!/usr/bin/env bash

# Ask for the administrator password upfront
sudo -v

# Keep-alive: update existing `sudo` time stamp until `.arch.bash` has finished
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &

sudo pacman -S \
  the_silver_searcher \
  awless \
  fzf \
  go \
  docker \
  docker-compose \
  neo-fetch \
  xsel \
  fzf \
  keybase \
  kbfs \
  ttf-fira-code \
  diff-so-fancy \
  python-pip \
  screen \
  git-lfs \
  jq \
  mtr \
  unzip \
  py3status

echo "Remove standard packages..."
sudo pacman -R \
    palemoon-bin \
    &> /dev/null
