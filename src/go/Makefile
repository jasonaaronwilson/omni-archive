all: build

go_binary = /usr/local/go/bin/go

build: core-archive-command.go
	${go_binary} build core-archive-command.go

format: *.go
	${go_binary} fmt *.go

test:	build
	rm -rf test-output
	mkdir test-output
	# create a simple archive
	./core-archive-command create test-output/test.car testdata/file1.txt testdata/file2.txt 
	# extract by filename
	(cd test-output && ../core-archive-command extract-files \
		test.car \
		testdata/file1.txt \
		testdata/file2.txt)
	cmp testdata/file1.txt test-output/testdata/file1.txt
	cmp testdata/file2.txt test-output/testdata/file2.txt
	# test the list command
	./core-archive-command list test-output/test.car > test-output/list.test
	cmp testdata/golden-list.test test-output/list.test
	rm -f test-output/testdata/file1.txt test-output/testdata/file2.txt 
	# test the extract-all command
	(cd test-output && ../core-archive-command extract-all test.car)
	cmp testdata/file1.txt test-output/testdata/file1.txt
	cmp testdata/file2.txt test-output/testdata/file2.txt

diff: clean format
	git difftool

clean:
	rm -rf test-output
	rm -f *~
	rm -f extract-output.txt
	rm -f output.txt
	rm -f core-archive-command
