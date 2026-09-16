#!/bin/sh
# hypridle condition_cmd: exit 0 (allow) unless the LG UltraGear is connected.
hyprctl monitors all -j | jq -e 'any(.[]; .description == "LG Electronics LG ULTRAGEAR 011NTLE3V064") | not' >/dev/null
