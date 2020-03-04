#!/usr/bin/env bash

# Terminate already running bar instances
killall -q polybar

# Wait until the processes have been shut down
while pgrep -u "$UID" -x polybar >/dev/null; do sleep 1; done


sleep .5

MONITOR="DP-1"


if ! pgrep -x polybar; then

	if type "xrandr"; then
		MONITOR=$MONITOR polybar -c "$HOME/.config/polybar/config.ini" base &
		for m in $(xrandr --query | grep " connected" | cut -d" " -f1); do
			MONITOR=$m polybar -c "$HOME"/.config/polybar/config.ini -l info bottom &
		done
	fi
else
	pkill -USR1 polybar
fi

echo "Bars launched..."
#   &> /home/dln/logs/polybar.log
