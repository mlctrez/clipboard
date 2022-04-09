
NAME=clipboard
HOST=salty

build: program/program.go
	mkdir -p temp
	go build -o temp/$(NAME) program/program.go

copy: build
	scp temp/$(NAME) $(HOST):/tmp/$(NAME)

deploy: copy
	ssh $(HOST) sudo /tmp/$(NAME) -action deploy

