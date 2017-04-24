FROM alpine:3.4
MAINTAINER Steve Louie <stelouie@cisco.com>

# create working directory
RUN mkdir -p /travis

# set the working directory
WORKDIR /travis

# add binary
COPY bin/travis /bin

# set the entrypoint
ENTRYPOINT ["/bin/travis"]
