export GOPATH = $(HOME)

target = github.com/rickyninja/tivo-archiver

all:
	go get -u $(target)
	go install $(target)
