#!/bin/bash

kubectl krew uninstall meshsync-snapshot 2>/dev/null
kubectl krew install --manifest=meshsync-snapshot.yaml --archive=bin/v0.0.1.tar.gz