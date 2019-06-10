
SHELL := /bin/bash

makefile_dir := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

install-all-packages:
	cat $(makefile_dir)packages.list | xargs apm install

packages-list:
	@rm -f $(makefile_dir)packages.list
	apm list --installed --bare | awk 'BEGIN {FS = "@"} {print $1}' > packages.list
