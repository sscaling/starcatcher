FROM alpine:latest
RUN apk add --no-cache git openssh
ADD .gitconfig /root/
ADD config /root/.ssh/
ADD robot_rsa /root/.ssh/id_rsa
ADD robot_rsa.pub /root/.ssh/id_rsa.pub
VOLUME ["/repo"]
WORKDIR /repo
