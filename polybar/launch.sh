#!/usr/bin/env sh

# Terminate already running bar instances
killall -q polybar

# Wait until the processes have been shut down
while pgrep -u $UID -x polybar >/dev/null; do sleep 1; done


sleep .5

MONITOR="HDMI1"


if ! pgrep -x polybar; then

	if type "xrandr"; then
		MONITOR=$m polybar -c $HOME/.config/polybar/config.ini base &> /home/dln/logs/polybar.log &
		for m in $(xrandr --query | grep " connected" | cut -d" " -f1); do
			# MONITOR=$m polybar --reload example &
			# MONITOR=$m polybar --reload example &
			# MONITOR=$m polybar -c $HOME/.config/polybar/config.ini secondary &> /home/dln/logs/polybar.log &
			MONITOR=$m polybar -c $HOME/.config/polybar/config.ini bottom &> /home/dln/logs/polybar.log &
		done
	else
		export MONITOR="HDMI1"
		polybar -c $HOME/.config/polybar/config.ini base &> /home/dln/logs/polybar.log &
		polybar -c $HOME/.config/polybar/config.ini bottom &> /home/dln/logs/polybar.log &
	fi
else
	pkill -USR1 polybar
fi

echo "Bars launched..."
