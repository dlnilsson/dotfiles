#!/usr/bin/env bash

# Terminate already running bar instances
killall -q polybar

# Wait until the processes have been shut down
while pgrep -u "$UID" -x polybar >/dev/null; do sleep 1; done


sleep .5

PRIMARY_MONITOR=$(xrandr --query | grep " primary" | cut -d" " -f1 )
MONITOR="DP-3"


if ! pgrep -x polybar; then

	if type "xrandr"; then
		# MONITOR=$MONITOR polybar -c "$HOME/.config/polybar/config.ini" base &
		MONITOR="DP-1" polybar -c "$HOME/.config/polybar/config.ini" secondary_top &
		MONITOR="DP-1" polybar -c "$HOME/.config/polybar/config.ini" secondary &
		MONITOR=$PRIMARY_MONITOR polybar -c "$HOME/.config/polybar/config.ini" base &
		MONITOR=$PRIMARY_MONITOR polybar -c "$HOME/.config/polybar/config.ini" bottom &
	fi
else
	pkill -USR1 polybar
fi

echo "Bars launched..."
#   &> /home/dln/logs/polybar.log
