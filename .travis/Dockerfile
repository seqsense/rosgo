ARG ROS_DOCKER=ros:kinetic-ros-base
FROM $ROS_DOCKER

VOLUME /usr/local/go

COPY . /go/src/github.com/seqsense/rosgo
COPY .travis/entrypoint.sh /entrypoint.sh

WORKDIR /go/src/github.com/seqsense/rosgo
CMD /entrypoint.sh
