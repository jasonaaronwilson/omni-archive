# Omni Archive File Format ("oar" file)

The "oar" file format (MIME type application/x-oar-archive), is a
redesign of the "tar" format and supports UTF-8 filenames without
jumping through any hoops. "oar" files use human readable (and human
editable) extensible *variable* length member headers rather than a
fixed length endian dependent format.

## Goals

The primary goals are the utmost simplicity and full support for
arbitrarily long UTF-8 encoded filenames without jumping through
hoops. "oar" files are so simple that even shell scripts can create
legal "oar" files without a library!

A much less important goal (which comes for free since we need
extensibility for future versions anyways) is the ability to have user
defined metadata making the omni archive format suitable as a
container format for more advanced use cases as a "container". [^1]

## Basic Format

An archive consists of zero or more members. (A completely empty "zero
length" file is a legal archive.)

A member consists of a variable sized header plus zero or more "raw"
bytes of binary data (aka the file contents).

A header *almost* looks like a UTF-8 encoded text file except that
instead of "newlines" to seperate the "lines" of the header, we use a
NUL byte (U+0000). For the rest of this document we will refer to
these as "lines" despite not ending in a traditional line seperator.

Each line is a UTF-8 string based key / value pair (so you might want
to read them into a string hashtable if you are implementing a
library). Keys may not contain "=" (U+003A) or NUL (U+0000) byte but
values are only restricted in that they may not contain the NUL byte
(U+0000).

A blank "line" is used to end the header (after which the data block,
if the size is > 0, is placed (otherwise it is the end of file or
another header).

Here is a sample header (where newlines are used instead of a NUL byte
to make it easier to present in this document):

```
filename=foo/var/baz/myfile.txt
size=1024

```

Again, where a line-break appears above it is actually not ASCII "10"
or "13", or but ASCII NUL (aka 0 aka U+0000).

It's canonical and recommended (but not required) to sort the header
by the keys. It is illegal to repeat a key in a header but some
workarounds are suggested in the extensibility section below.

## Extensibility

Keys that begin with "x-" are meant to be used for additional
non-standard metadata that are specific to certain applications. Tools
should preserve this metadata unless the user requests they be
removed.

In terms of extensibility, here is a sample header with user defined
meta-data as a simple key/value pair, suitable for most use cases
where only a small amount of additional meta-data is required:

```
filename=foo/var/baz/myfile.txt
size=1024
x-my-application-part-type=primary-icon
x-my-application-foo-key=baz

```

If you don't like key value pairs, then you can just encode your
application specific metadata using any text based encoding such as
XML, JSON, TOML, etc. as long as they are valid UTF-8 and don't
include a NUL byte:

```
filename=foo/var/baz/myfile.txt
size=1024
x-my-application-json-metadata={
  version: 100,
  name: "foo",
  offsets: [100, 897, 3678],
}
x-another-custom-key=whatever

```

There is no extra code required in the archive utility to understand
anything about JSON to process (and thus retain) this header and
delimiters like "{" and "}" are not treated specially by the archive
tool. What is difficult to see, because the examples shown here
conflate line breaks (aka newlines) and the NUL byte, is that
x-my-application-json-metadata uses real "newlines" internally while
the actual value string associated with the key
x-my-application-json-metadata is like any other key that ends with a
NUL byte right after the closing "}" - it just happens to also contain
internal newlines so it can look pretty.

Let's do the above example again. This time line-breaks are just part
of the presentation and we will explicitly use the strings (U+0000)
and (U+000A) to represent the real seperator byte (since headers are
always UTF-8 we know these are just one byte long):

```
filename=foo/var/baz/myfile.txt(U+0000)
size=1042(U+0000)
x-json-metadata={(U+000A)
  version: 100,(U+000A)
  name: "foo",(U+000A)
  offsets: [100, 897, 3678],(U+000A)
}(U+0000)
x-another-custom-key=whatever(U+0000)
(U+0000)
```

Since it is illegal to repeat a key in a header. You might want to use
this format for certain keys that are array like instead:

```
x-my-array.0=...
x-my-array.1=...
```

And if you need "maps", then maybe this suffices:

```
x-my-map.foo=...
x-my-map.bar=...
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
