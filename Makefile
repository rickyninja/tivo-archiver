export GOPATH = $(CURDIR)
export CGO_ENABLED = 0

target = tivo-archiver

all:
#	go install tvrage
#	go install -a -installsuffix cgo $(target)
	go get $(target)
	go build -o bin/$(target) -a -installsuffix cgo $(target)
