#!/bin/bash

go build -ldflags="-s -w" -trimpath -o hometrustd .
sudo cp hometrustd /usr/local/bin
sudo cp hometrustd.service /etc/systemd/user/
sudo cp hometrustd.service /etc/systemd/system/hometrustd.service # ~/.config/systemd/user/myservice.service
systemctl --user daemon-reload
