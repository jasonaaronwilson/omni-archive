all: build

go_binary = $(shell { command -v go || command -v /usr/bin/go || command -v /usr/local-bin/go; } 2>/dev/null)

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
	(cd test-output && ../core-archive-command extract-by-file-name \
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
	(cd test-output && ../core-archive-command extract test.car)
	cmp testdata/file1.txt test-output/testdata/file1.txt
	cmp testdata/file2.txt test-output/testdata/file2.txt
	# test the append command
	./core-archive-command create test-output/test-two.car testdata/file3.txt testdata/file4.txt
	./core-archive-command append test-output/append.car test-output/test.car test-output/test-two.car
	rm -f test-output/testdata/file1.txt \
	      test-output/testdata/file2.txt \
	      test-output/testdata/file3.txt \
	      test-output/testdata/file4.txt
	(cd test-output && ../core-archive-command extract append.car)
	cmp testdata/file1.txt test-output/testdata/file1.txt
	cmp testdata/file2.txt test-output/testdata/file2.txt
	cmp testdata/file3.txt test-output/testdata/file3.txt
	cmp testdata/file4.txt test-output/testdata/file4.txt
	(cd test-output && ../core-archive-command remove-by-file-name removed.car append.car testdata/file3.txt)
	./core-archive-command list test-output/removed.car > test-output/removed-list.test
	cmp testdata/golden-removed-list.test test-output/removed-list.test
	# test create archive from a directory
	./core-archive-command create test-output/all-testdata.car testdata
	./core-archive-command list test-output/all-testdata.car > test-output/all-list.test
	cmp testdata/golden-all-list.test test-output/all-list.test

diff: clean format
	git difftool

clean:
	rm -rf test-output
	rm -f *~
	rm -f testdata/*~
	rm -f extract-output.txt
	rm -f output.txt
	rm -f core-archive-command
