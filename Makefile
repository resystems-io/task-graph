GOSRC=$(shell find . -name '*.go')
TMPLSRC=$(shell find . -name '*.tmpl')

build/task-graph: $(GOSRC) $(TMPLSRC)
	go build -o build/ ./...

run: build/task-graph
	./build/task-graph -o resystems-io -r task-graph -n 1 mermaid

test:
	go test ./...

install:
	go install ./...

upgrade:
	go get -u ./...

clean:
	@rm -rf build

.PHONY: test install clean run upgrade
