#!/bin/bash

go build ./app/main.go
sudo chown root:root main
sudo chmod u+s main
sudo mv main /usr/bin/vcs