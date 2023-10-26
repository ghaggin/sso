#!/bin/bash

[[ -z "$1" ]] && echo "please provide a name" && exit 1
[[ -z "$2" ]] && echo "please provide a subject" && exit 2

DIR="../keys"

openssl req -x509 -newkey rsa:2048 -keyout "$DIR/$1.key" -out "$DIR/$1.cert" -days 365 -nodes -subj "/CN=$2"
