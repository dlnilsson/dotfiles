#!/bin/sh
# Allow idle actions unless the LG UltraGear is connected on the "tardis" wifi.
hyprctl monitors all -j | jq -e 'any(.[]; .serial == "011NTLE3V064") | not' >/dev/null && exit 0
[ "$(iwgetid -r)" != "tardis" ]
