

alias docker-nah "docker ps -a | awk '{ if ($1 != "CONTAINER") system("docker kill " $1) system("docker rm " $1) }'"
