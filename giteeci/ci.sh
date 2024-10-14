#!/bin/bash

docker build --build-arg GH_TOKEN=${GH_TOKEN} --build-arg GH_USER=${GH_USER} --build-arg GITEE_USER=${GT_USER} --build-arg GITEE_TOKEN=${GT_TOKEN} -t gitea -f Dockerfile.rootless_cdn .
