all: run

go_binary = /usr/local/go/bin/go

build: core-archive-command.go
	${go_binary} build core-archive-command.go

format: core-archive-command.go
	${go_binary} fmt core-archive-command.go

clean:
	rm -f *~
	rm -f extract-output.txt
	rm -f output.txt
	rm -f core-archive-command





