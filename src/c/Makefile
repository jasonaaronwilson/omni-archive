.PHONY all: build

build:
	gcc -g -rdynamic -o oarchive $(realpath oarchive.c)

clean:
	rm -f oarchive *~

format:
	clang-format -i *.c
