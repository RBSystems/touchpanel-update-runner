#!/usr/bin/env bash
# Pasted into Jenkins to build (will eventually be fleshed out to work with a Docker Hub and Amazon AWS)

echo "Stopping running application"
docker stop touchpanel-update-runner
docker rm touchpanel-update-runner

echo "Building container"
docker build -t byuoitav/touchpanel-update-runner .

echo "Starting the new version"
docker run -d --restart=always --name touchpanel-update-runner -p 8004:8004 byuoitav/touchpanel-update-runner:latest
