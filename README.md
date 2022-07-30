# Core Archive File Format (car file)

The "car" file format (MIME type application/x-core-archive), is a
redesign of the "ar" format which uses human readable variable length
headers (which are used to store the size, filename, and other
metadata associated with each file) and raw data for the file content
themselves.

Core archive supports aligning the raw data, checksums on the raw
data, data compression of the raw data, ridicuously long unicode
file-names, posix file metadata, application specific text metadata,
and application specific binary meta-data such as indexes, symbol
tables, etc. It's pureposefully meant to be embraced and extended
while the primary feature of storing blobs of data with associated
file names is preserved.

## The Format

A "car" file is a sequence of "members" (where usually but not always
a member represents a file). Each member has a human readable header
followed by a binary blob of data of zero or more bytes (the "size:
comes from the header).

It's legal to simply "cat" together "car" files to create a combined
archive (though this may invalidate some application specific
metadata).

Unlike "ar" files, car files do not currently have a magic
number. (Maybe they should - could look like "car: 1\n" for example.)

### Header Format

The header is a series of unix style utf-8 encoded lines (each line
ends with U+000A). A single empty line ends the header (though
additional newlines are used as padding bytes to reach the alignment
for the begining of the data when the alignment is not 1).

There isn't a mechanism to force alignment of headers (the "ar" format
aligns headers and data to 16bit boundaries). Since headers don't
contain binary data, this should not be a problem.

Each unix style line of a member header is a series of keys and values
in the following format:

<key in utf8>: <value in utf8>\n

The space trailing after the ": is" *required* and only one space is
allowed.

Keys are arbitrary sequences of unicode utf8 characters that don't
contain U+0020, U+003A, U+0000, or U+000A. However, all of the
standard keys are 7-bit ASCII printable characters (a subset of
unicode utf8).

Values are either base-10 integers (possibly with a leading "-") or
are meant to be treated as utf-8 strings (for example to hold the file
name metadata). The U+0000 and U+000A are not currently allowed in key
values.

When parsing values to integers, readers should handle all numbers
representable in a 64bit 2's complement signed number. "-0" should be
mapped to zero (though implementations should not use "-0").

The only key that must be present in the header is "size:" though the
size is allowed to be zero if there is no data contents (though the
header must still be right padded to the alignment using \n after the
blank line used to indicate the end of the header).

The nearly full set of well-known keys with some sample values:

```
align: 1
copyright: Copyright (c) 2022 Jason A. Wilson.
data-compression-algorithm: application/gzip
data-hash-algorithm: SHA-256
data-hash: 784f6696040e7a4eb1465dacfaf421a526d2dd226601c0de59d7a1b711d17b99
data-size: 3024
file-name: foo.txt
file-version: 17
license: https://www.gnu.org/licenses/gpl-3.0.en.html
mime-type: text/plain
posix-file-mode: -rw-r--r--
posix-group-id: 100
posix-group-name: jawilson
posix-modification-time: 2022-10-25 18:37:43.962000000
posix-owner-id: 100
posix-owner-name: jawilson
size: 1876
```

This isn't a valid header because it *doesn't end in a blank line* but
otherwise I hope this should give a clear sense of how things are
represented.

TODO(jawilson): add entries for each standard key and there values
(especially the posix-modification-time).

TODO(jawilson): file ACLs and other file metadata? Data from MacOS or
Windows?

Without compression, size and data-size must be exactly the same (and
therefore data-size should be ignored).

The data-hash should be computed pre-compression.

Keys that begin with "x-" are not described in this document and are
meant to be used for small amounts of non-standard metadata that are
not described in this document. Tools should leave this metadata alone
unless the user requests they be removed.

It is sometimes legal to repeat keys with different values. For the
standard keys defined above, the first key value pair should be used
and for those keys, other key value pairs with the same key can be
dropped.

It is *recommended* that alignment *not* be set for compressed members
(or simply use the default "align: 1\n".

Implementations can expect all "lines" in the header to be <= 1023
bytes long though most are expected to be fairly short (<= 80 bytes).

### Member Data Format

As noted, the member data can be aligned (based on the offset with-in
the archive file modulo the alignment). Additionally, the member data
should be padded to the appropriate alignment as well by using
trailing zero bytes to match the alignment. The rationale is to allow
car files to exist that have all data aligned to architecture specific
page boundaries for reading the data without reading data for a header
that is unrelated to that entry.

Otherwise, the member data is just the raw data bytes in a file, a
compressed version of the raw data using the given compression
algorithm, or some application specific data (for example, indexes of
various sorts).

We only recommend using application/gzip for general purpose core
archive files since that is very widely available. compression is
turned off by default (one can always compress the entire "car"
archive of course though not all tools would understand that without
first decompressing especially if the compression algorithm isn't
application/gzip).

## Indexes, Symbol Tables, Etc.

It may be desireable to store additional metadata in an efficient
binary format that either describes the entire archive file itself or
to specific files in the archive.

When a header is describing such a metadata blob, it can omit the
"file-name:" key and add the line "binary-metadata: true\n" so other
tools know this isn't data isn't necessarily needs to be extracted.

When this binary metadata is about a particular file in the archive,
the key "for-file-name:" can be used (multiple times, the only
standard header that works this way).

The size: attribute must still be set as always and they mime-type:
should also be set to something appropriate.

If "for-file-name:" key is not used and the appropriate mime-type is
insufficient to locate the desired additional metadata, then the key
"identifier:" can be used.

We also recommend using one or more additional key/value pairs so that
a consumer of this index can determine if it is up to date or not (the
exact recommendation is TBD) since tools that manipulate a car file
could add new files without updating this application specific
metadata.

## Multiple Entries with the same file-name and Version Numbers

The car format allows multiple members with the same file-name as long
as they have different version numbers. By default, the highest
versioned member should be "returned" when requesting a member by name
when using a care file reader library without a version
number. Naturally some archives will be produced that have the same
file-name but no version information. In this case the library should
provide a way to get the first occurence and ignore the other member
(though it can indicate a warning condition).

## Lite Archives

"ar" provides a format whereby only metadata is supplied, and the data
contents are empty and are expected to be found in the file-system
(somehow). To produce such archives, one merely needs to set size: to
0 and add "is-lite: true\n" to the header though the "ar" equivalent
requires that all such members are lite.

# Discussion

core archive files will likely have a slightly larger foot-print than
"ar" files because of the uncompressed fully human readable member
headers ("ar" also does not compress it's headers and because of their
fixed size, they have caused great confusion regarding long file names
and unicode characters in file names).

The only unicode code-points that aren't allowed in file names are
U+0000 and U+000A. A quick Google search suggests that U+000A is in
fact allowed in file-names on some operating systems. I have several
solutions to this problem.

Consideration was made for using LEB128, especially for size headers,
rather than base 10 human readable strings but was rejected (the
format itself doesn't dis-allow this, only that implementations are
currently allowed to make some assumptions to simplify their current
implementations).

# Implementations

The first implementation of a "car" file tool will be using bash plus
expected binaries on a unix/linux implementation to generate a legal
"car" file or append "car" files to make larger archives (without
handling duplicate file-name and such).

# Test Suite

TBD.

Some of the most interesting cases, such as handling archives larger
than 32bits may not be practical to provide.

# Pull Requests

I plan to implement a very basic bash version of a tool to create
archives, "join" them, and extract them. It will not have the
capability of deleting members, etc. This will serves as a proof of
concept.

I also plan to create a Java library that will read and write "car"
files.

I also plan to create a Go crate that will read and write "car" files.

I may eventually create a C file that can read and write "car" files.

However, that will still leave many languages without a native library
that can manipulate "car" files. The point of a general purpose
archive format is that they can be universally implemented and I will
be very interested in help from expert coders in those languages.

A FUSE file-system would be a very interesting so that a "car" file
could be "mounted".

# Conclusion

core archive files (".car" files) are a general purpose way of storing
multiple files, their meta-data, and even meta-data and indexing
information about the archive itself. If data-compression is not used,
then very simple libraries should be possible to allow any program to
read and especially write core archive files.

