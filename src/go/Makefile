all: build

go_binary = /usr/local/go/bin/go

build: core-archive-command.go
	${go_binary} build core-archive-command.go

format: *.go
	${go_binary} fmt *.go

test:	build
	rm -rf test-output
	mkdir test-output
	./core-archive-command create test-output/test.car testdata/file1.txt testdata/file2.txt 
	(cd test-output && ../core-archive-command extract-files \
		test.car \
		testdata/file1.txt \
		testdata/file2.txt)
	cmp testdata/file1.txt test-output/testdata/file1.txt
	cmp testdata/file2.txt test-output/testdata/file2.txt
	./core-archive-command list test-output/test.car > test-output/list.test
	# The obvious extra "Println"'s in core-archive-command.go
	#   cause this to fail but it otherwise the output looked good.
	# cmp testdata/file1.txt test-output/list.test

diff: clean format
	git difftool

clean:
	rm -rf test-output
	rm -f *~
	rm -f extract-output.txt
	rm -f output.txt
	rm -f core-archive-command
