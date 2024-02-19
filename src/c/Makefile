.PHONY all: build

build:
	gcc -g -rdynamic -o oarchive $(realpath oarchive.c)

clean:
	rm -rf oarchive *~ test-out

test:
	./test.sh

format:
	clang-format -i *.c
