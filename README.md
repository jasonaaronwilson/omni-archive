# Omni Archive File Format ("oar" file)

The "oar" file format (MIME type application/x-oar-archive), is a
redesign of the "tar" format and supports "infinitely" long UTF-8
filenames without jumping through any hoops and supports user defined
metadata.

"oar" files use human readable *variable* length headers rather than a
fixed length "binary" endian dependent format. The "oar" file format
supports forwards and backwards compatibility.

## Example Omni Archive File

Archives can be extremely simple like the example below that contains
two "files" named `hello.txt` and `world.txt`. Any line breaks shown
here are simply for presentation purposes and do not denote any
encoding of a "newline". "\0" is used to represent U+00000, NUL, '\0',
0, 0x0, or whatever you want to call the zero byte. Everything else
happens to be 7-bit ASCII though headers can use UTF-8 (and must be
compatible with it).

```
size=5\0
file=hello.txt\0
\0
HELLOfile=world.txt\0
size=6\0
\0
WORLD!\0
```

A detailed breakdown of this appears below.

## Omni Archive File Format Specification

An archive consists of an optional "magic number" (which can be
treated as a user defined key/value pair) and then zero or more
members. (A completely empty "zero length" file is a legal archive.)

### Magic Number

"magic numbers" are unique byte sequences at the beginning of a file
that identify its type or format. While this is completly optional,
archive tools (when creating from scratch) will generally add this
ASCII sequence to the top of an archive file:

```
x-OR=magic\0
```

Since this is formatted like a key/value pair, read only tools will
generally not have to do anything special to deal with magic numbers.

### Members

A member (aka file) consists of a variable sized header plus zero or
more "raw" bytes of binary data (aka the data or file contents).

A header *almost* looks like a UTF-8 encoded text file except that
instead of "newlines" to seperate the "lines" of the header, we use a
NUL byte (U+0000). For the rest of this document we will refer to
these as "lines" despite not ending in a traditional line seperator.

Each line is a UTF-8 string based key / value pair (which makes them
compatible with a sting -> string hashtable if you are implementing a
library). The end seperator of a key is "=" (U+003A) and therefore the
key may not contain this character (or NUL), however value strings are
only restricted in that they may not contain NUL/(U+0000). Unlike many
formats, there is no mechanism to quote either U+003A or U+0000 which
also means that parsers and printers can be extraordinarily simple.

A blank "line" is used to end the header.

Immediately afterwords "size" bytes of binary bytes must be present
because after the header comes the raw data block which may be absent
when the "size" key is absent or when decoded as a base 10 integer is
zero). If the size is missing or zero, then another header or the end
of file is immediately adjacent.

Here is a sample header again using \0 to denote U+0000 and inserting
line-breaks purely for presentation value.

```
filename=foo/var/baz/myfile.txt\0
size=1024\0
\0
```

It is illegal to repeat a key in a header but some workarounds are
suggested in the extensibility section below.

## Additional Standard Key Value pairs

(tool support coming soon...)

### MIME Types

It may be useful to store a MIME type to describe what the binary data
blob holds (most filesystems don't have a way to store this so they
typically will not survive a round-trip through extraction to the
filesystem and re-archiving, just like user defined keys).

Example:

```
mime-type=text/plain
```

### Checksums

One obvious missing feature for an archive tool are checksums. We
define two additional keys to allow for the integrity of the binary
data in individual members though these are not added or checked by
default (default checking will eventually become the default for
tools):

```
data-hash-algorithm:SHA-256
data-hash:784f6696040e7a4eb1465dacfaf421a526d2dd226601c0de59d7a1b711d17b99
```

### POSIX file information

In order to fully reconstruct a file on disk the way it was when the
archive was first created, we can include additional metadata:

```
posix-file-mode=-rw-r--r--
posix-group-name=jawilson
posix-group-number=100
posix-modification-time-nanos=112000000
posix-modification-time-seconds=23486345
posix-owner-name=jawilson
posix-owner-number=100
```

TODO(jawilson): is this complete? what about creation time?

The modification time is split into two 64bit fields to simplify usage
with languages that lack support for manipulating integers above
64bits.

## Unsupported Features

* alignment of the data of a member to play nicely with mmap
* indexes for fast random access to individual files + archive wide
  checksums (we will define a standard simple way to do both in a
  future version).
* data compression of members - tar also does not support compression
  which is why "tar" file are typically gzipped though this also makes
  random access "impossible"

## Standard Tools

The `oarchive` tool can be used to create, list, append, and extract
members (technically "cat" can be used to append two archives but
append might do smart things later like recreate indxes and maintain
file-wide checksums).

`oarchive` actually has two implementations, one in Go, and one in
standard C for platforms not supported by Go or in cases where the
smallest toolsize is beneficial.

Here are some sample invocations of `oarchive`

```
oarchive create --output-file=output.oar --verbose=true file1.txt file2.jpg
oarchive extract --input-file=input.oar
oarchive extract --input-file=input.oar --output-directory=/tmp/foo
oarchive list --input-file=input.oar
oarchive append --output-file=output.oar archive1.oar archive2.oar
cat foo.oar | oarchive extract
```

This is definitely not as terse as other tools though shell aliases,
shell functions, or shell script wrappers can easily make the archive
command line act more like "tar", "ar", "zip", etc. according to your
preference.

# Discussion

Omni archive files may have a slightly larger foot-print than "ar"
files because of the uncompressed semi human readable member headers
("ar" headers are fixed size and because of this they have caused
great confusion regarding long file names and unicode characters in
file names).

I considered a different format for values, namely, C/Java/Javascript
style strings using U+005C as an escape sequence (and of course
supporting \uXXXX to retain full unicode support). That would have
required much more logic in all the libraries that process these
values.

# Conclusion

The oar archive file format is an extensible archive format that is
extremely easy to produce and consume even without a special
library. That they support UTF-8 long filenames with minimal kludges
is a huge plus.


TODO(jawilson): determinism...

Tools should be deterministic. There is not an easy to describe
algorithm for this since many technique will work however

so that it can be part of a
"reproducible build". One way to do this would be to sort the header
lines by the key but make sure that magic numbers remains at the top
of the file.



## Example Breakdown

This may look scary but it is very easy to handle if you are
processing "oar" files. 



Looking at it with the \0 removed but with a
similar presentation as above yields:

```
size=5
file=hello.txt

HELLOfile=world.txt
size=6

WORLD!
```

In the above example "HELLO" could have been any 8-bit binary data
(just make sure you also adjust size=5 to reflect the "payload").

"HELLOfile" looks weird but that's just because there is nothing that
appears after a binary blob to seperate it from the start of the next
header. "size=6" might seem off by one but that's because you might
think the extra U+0000 between the "6" and the "W" is part of the
binary data for the member (it's actually part of the header, aka, the
blank "line" that terminates the header).



Archive tools should keep this 

Additionally, archive tools will sort the metadata "lines" such that
keys that start with "x-" and have a value of "magic" will stay at the
top of a file. This allows you to define your own "magic number" but
still be compatible with any tool that can read omni archive files.

