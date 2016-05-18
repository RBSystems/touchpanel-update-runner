FROM golang:1.6

RUN mkdir -p /go/src/github.com/byuoitav
ADD . /go/src/github.com/byuoitav/touchpanel-update-runner

WORKDIR /go/src/github.com/byuoitav/touchpanel-update-runner
RUN go get -d -v
RUN go install -v

CMD ["/go/bin/touchpanel-update-runner"]

EXPOSE 8004
