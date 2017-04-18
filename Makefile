export GOPATH = $(HOME)

target = github.com/rickyninja/tivo-archiver

all:
	go get $(target)
	go install $(target)
