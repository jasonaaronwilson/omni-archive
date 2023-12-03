# Omni Archive File Format ("oar" file)

The "oar" file format (MIME type application/x-oar-archive), is a
redesign of the "tar" format and supports UTF-8 filenames without
jumping through any hoops with absolute simplicity as the main
goal. "oar" files use human readable *variable* length member headers
rather than a fixed length, endian dependent, format so it can also be
easily extended with a large degree of backwards and forwards
compatbility.

## Sample Omni Archive File

```
file:hello.txt\0
size:5\0
\0
HELLOfile:world.txt\0
size:6\0
\0
WORLD!\0
```

## Goals

The primary goals are the utmost simplicity and full support for
arbitrarily long UTF-8 encoded filenames without jumping through
hoops. "oar" files are so simple that even shell scripts can create
legal "oar" files without a library. (TODO(jawilson): create a bash
script that creates a legal ".oar" file from it's arguments and link
to it here.)

A much less important goal (which comes for free since we need
extensibility for future versions anyways) is the ability to have user
defined metadata making the omni archive format very suitable as a
container format. [^1]

## Basic Format

An archive consists of an optional "magic number" (which can be
treated as a user defined key/value pair) and then zero or more
members. (A completely empty "zero length" file is a legal archive.)

### Magic Number

Magic numbers are unique byte sequences at the beginning of a file
that identify its type or format. These are completly optional but
archive tools will generally add this ASCII sequence to the top of the
file:

```
x-OR=magic\0
```

### Members

A member consists of a variable sized header plus zero or more "raw"
bytes of binary data (aka the data or file contents).

A header *almost* looks like a UTF-8 encoded text file except that
instead of "newlines" to seperate the "lines" of the header, we use a
NUL byte (U+0000). For the rest of this document we will refer to
these as "lines" despite not ending in a traditional line seperator.

Each line is a UTF-8 string based key / value pair (which makes them
suitable to treat as a sting hashtable if you are implementing a
library with deep manipulation skills). The seperator is "=" (U+003A)
and the key may not contain this characteer (or NUL/(U+0000)), however
value strings are only restricted in that they may not contain
NUL/(U+0000). Unlike many formats, there is no mechanism to quote
either U+003A or U+0000 which also means that parsers and printers can
be extraordinarily simple.

A blank "line" is used to end the header.

After the header comes the raw data block (when the size string
interpreted as a base 10 integer is > 0). If the size is missing or
zero, then another header or the end of file is immediately adjacent.

Here is a sample header again using \0 to denote U+0000 and inserting
line-breaks purely for presentation value.

```
filename=foo/var/baz/myfile.txt\0
size=1024\0
\0
```

It's higly recommended that your tool be deterministic so that it can
be part of a "reproducible build". One way to do this would be to sort
the header lines by the key.

It is illegal to repeat a key in a header but some workarounds are
suggested in the extensibility section below.

## Extensibility

Keys that begin with "x-" are meant to be used for additional
non-standard metadata. Tools should preserve this metadata unless the
user requests they be removed.

Example:

```
filename=foo/var/baz/myfile.txt\0
size=1024\0
x-my-application-part-type=primary-icon\0
x-my-application-foo-key=baz\0
\0
```

If you find this limiting, then you can just encode your application
specific metadata using any text based encoding such as XML, JSON,
TOML, etc. as long as they are valid UTF-8 and don't include a NUL
byte:

```
filename=foo/var/baz/myfile.txt\0
size=1024\0
x-my-application-json-metadata={\n
  version: 100,\n
  name: "foo",\n
  offsets: [100, 897, 3678],\n
}\0
x-another-custom-key=whatever\0
\0
```

There is no extra code required in the archive utility to understand
anything about JSON to process (and thus retain) this header and
delimiters like "{" and "}" are not treated specially by the archive
tool. 

Since it is illegal to repeat a key in a header. You might want to use
this format for certain keys that are array like instead:

```
x-my-array/0=...\0
x-my-array/1=...\0
```

And if you need "maps", then maybe this suffices:

```
x-my-map/foo=almost anything...\0
x-my-map/bar=could be put here except NUL...\0
```

If you want to organize your metadata more, you could use "." as part
of your keys:

```
x-com.google.archive.notes.word-wrap=false\0
x-com.google.archive.foo.bar=false\0
```

### Magic Numbers

If an omni archive file starts with "x-" and eventually has "=\0" then
it must be treated as a user extension and is thus legal to place at
the beginning of the file (if the file isn't empty?). Just make sure
the entire sequence is legally encoded UTF-8.


tools must already support this as input essentially like a
comment. There is one restriction and that is that the entire magic
number must still be a valid UTF-8 string.


Magic numbers are a byte sequence at the beginning of a file that
identify the type or format. Classically these were 32bits but many
popular formats like JPEG 2000, HEVC, WebP, AVIF, 

While oarchive doesn't have a "magic number" (i.e., bytes at the
beginning of a file that give a strong indication of a particular kind
of file), you can still place a subset of magic numbers at the very
begining of an archive yourself.

In order for you magic number to adhere to the specification, it only
needs to begin with "x-" and be valid UTF-8. Then you *must* also make
sure that there is at least "=\0" after it though we prefer you append
"=magic\0" after it since that may be helpful to both users and tool.

Essentially all new file formats have tran

For a "more" unique magic number (which I suggest), place this at the
begging of the file:

```
x-YYYYYY=magic
```

Where Y is any ASCII character except '=' or NUL.

```
x-bb=magic
```

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
posix-file-mode:-rw-r--r--
posix-group-name:jawilson
posix-group-number:100
posix-modification-time-nanos:112000000
posix-modification-time-seconds:23486345
posix-owner-name:jawilson
posix-owner-number:100
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
  which is why they are typically gzipped though this also makes
  random access impossible

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

[^1] For example, if you were designing a file format for a word
processor, you might store the document text as one "logical"
file-name (maybe in XML?) and then every image in the document could
be stored as other "logical" file-names (presumably PNGs, JPGs, GIFs,
etc.) Maybe you really wanted each "chapter" to be it's own logical
file, you can do that too! Another example, if you wanted to create
something like a "web archive", perhaps each HTML, JS, and image files
could have a property like x-source-url to keep track of where these
were obtained.
