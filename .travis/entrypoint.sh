#!/bin/bash

set -e

source /opt/ros/${ROS_DISTRO}/setup.bash
export PATH=/go/bin:/usr/local/go/bin:$PATH
export GOPATH=/go

roscore &

go install github.com/seqsense/rosgo/gengo
go generate ./test/test_message
go test ./xmlrpc -v
go test ./ros -v
go test ./test/test_message -v
go test -tags integration . -v
