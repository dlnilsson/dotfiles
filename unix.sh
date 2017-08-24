#!/bin/bash
#
# Security checks and other goodies
#
# https://github.com/drduh/macOS-Security-and-Privacy-Guide#hosts-file
#

cmn_init() {
    # Will exit script if we would use an uninitialised variable:
    set -o nounset
    # Will exit script when a simple command (not a control structure) fails:
    set -o errexit

    set -e
}

cmn_assert_running_as_root() {
    if [[ ${EUID} -ne 0 ]]; then
        cmn_die "This script must be run as root!"
    fi
}

info() {
    local green=$(tput setaf 2)
    local reset=$(tput sgr0)
    echo -e "${green}$@${reset}"
}

notice() {
    local blue=$(tput setaf 4)
    local reset=$(tput sgr0)
    echo -e "${blue}$@${reset}"
}

important() {
    local yellow=$(tput setaf 3)
    local reset=$(tput sgr0)
    echo -e "${yellow}$@${reset}"
}

cmn_die() {
    local red=$(tput setaf 1)
    local reset=$(tput sgr0)
    echo >&2 -e "${red}$@${reset}"
    exit 1
}

confirm() {
    # call with a prompt string or use a default
    read -r -p "${1:-Are you sure? [y/N]} " response
    case "$response" in
        [yY][eE][sS]|[yY])
            true
            ;;
        *)
            false
            ;;
    esac
}

main() {
	cmn_assert_running_as_root
	dns_hostfile
}

dns_hostfile() {
	local count="$(wc -l /etc/hosts | awk '{print $1}')"
	info "add blacklisted host to /etc/hosts"
	info "you've $count right now in /etc/hosts"
	confirm && curl "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts" | sudo tee -a /etc/hosts
	info "found $(wc -l /etc/hosts | awk '{print $1}') row"
}


main
