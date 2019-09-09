#!/usr/bin/env bash

# Ask for the administrator password upfront
sudo -v

# Keep-alive: update existing `sudo` time stamp until `.arch.bash` has finished
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &

# update
sudo pacman -Syy

sudo pacman -S --noconfirm the_silver_searcher
sudo pacman -S --noconfirm awless
sudo pacman -S --noconfirm fzf
sudo pacman -S --noconfirm go
sudo pacman -S --noconfirm docker
sudo pacman -S --noconfirm docker-compose
sudo pacman -S --noconfirm neofetch
sudo pacman -S --noconfirm xsel
sudo pacman -S --noconfirm fzf
sudo pacman -S --noconfirm keybase
sudo pacman -S --noconfirm kbfs
sudo pacman -S --noconfirm ttf-fira-code
sudo pacman -S --noconfirm diff-so-fancy
sudo pacman -S --noconfirm python-pip
sudo pacman -S --noconfirm screen
sudo pacman -S --noconfirm git-lfs
sudo pacman -S --noconfirm jq
sudo pacman -S --noconfirm mtr
sudo pacman -S --noconfirm unzip
sudo pacman -S --noconfirm py3status
sudo pacman -S --noconfirm termite
sudo pacman -S --noconfirm polybar
sudo pacman -S --noconfirm rofi
sudo pacman -S --noconfirm bind-tools
sudo pacman -S --noconfirm adobe-source-code-pro-fonts
sudo pacman -S --noconfirm noto-fonts-emoji
sudo pacman -S --noconfirm ttf-nerd-fonts-symbol
sudo pacman -S --noconfirm gnu-netcat
sudo pacman -S --noconfirm whois
sudo pacman -S --noconfirm exa
sudo pacman -S --noconfirm imagemagick
sudo pacman -S --noconfirm copyq
sudo pacman -S --noconfirm kitty
sudo pacman -S --noconfirm tilix
# python-gnupg


yay ttf-font-awesome-4
yay termsyn-font
#yay ttf-iosevka
yay feh
yay betterlockscreen
yay uthash
yay deadd-notification-center
yay notify-send.sh
yay wpgtk

echo "Remove standard packages..."
sudo pacman -R \
    palemoon-bin \
    &> /dev/null
