#!/usr/bin/env bash

# Ask for the administrator password upfront
sudo -v

# Keep-alive: update existing `sudo` time stamp until `.arch.bash` has finished
while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null &

echo "Updating pacman..."
sudo pacman -Syy

arch_packages="$HOME/.dotfiles/arch_packages"
echo "Installing packages from $arch_packages"
while IFS= read -r pkg; do
	sudo pacman -S --noconfirm  "$pkg"
done <"$arch_packages"

aur_packages="$HOME/.dotfiles/aur_packages"
echo "Installing AUR packages from $aur_packages"
while IFS= read -r pkg; do
	yay "$pkg"
done <"$aur_packages"

echo "Remove standard packages..."
sudo pacman -R \
    palemoon-bin \
    &> /dev/null
