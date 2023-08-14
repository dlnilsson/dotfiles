#!/usr/bin/env bash
# See: https://gist.github.com/Djuuu/0a576ec006ba23c24c9576d6a36baaed
#
# Horizontal scroll sensitivity reduction
#
hScrollModulo=3
hScrollIndexBuffer="/dev/shm/LogitechMXMaster3HScroll"


# Create temporarily file if it doesn't already exist
if [ ! -f "$hScrollIndexBuffer" ]; then
    printf "L\n0\n" > "$hScrollIndexBuffer"
fi


function temporizeHorizontalScroll {
  local newDirection=$*;

  # read buffer
  local buffer=($(cat $hScrollIndexBuffer))
  local oldDirection=${buffer[0]}
  local value=${buffer[1]}

  if [ "$oldDirection" = "$newDirection" ]; then
    # increment
    ((value++))
    ((value%="$hScrollModulo"))
  else
    # reset on direction change
    value=1
  fi

  # write buffer
  echo "$newDirection $value" > $hScrollIndexBuffer || value=0

  # temporize scroll
  [ ${value} -ne 0 ] && exit;
}


function main {
  local button=$1
  case $button in
    "Scroll_L")
      temporizeHorizontalScroll "L"
      xdotool key --clearmodifiers ctrl+shift+Tab; ;; # Previous tab
    "Scroll_R")
      temporizeHorizontalScroll "R"
      xdotool key --clearmodifiers ctrl+Tab; ;; # Next tab
  esac
}

main "$1"


