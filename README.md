# Core Archive File Format (car file)

The "car" file format (MIME type application/x-core-archive), is a
redesign of the (various) "ar" formats and "car" uses fully human
readable variable length member headers rather than a format partially
designed just to be "easy" for C to handle.

Member headers are used to store metadata about the file contents such
as the size (required), filename, and other metadata normally
associated with an individual file in a file system and of course the
raw data content of each member.

The core archive file format supports:

1. aligning raw data (for example to page sizes)
2. checksums on the raw data (to detect corruption)
3. data compression of the raw data
4. ridicuously long unicode file-names (technically unlimited)
5. posix file metadata
6. simple application specific metadata extensions
7. application specific binary meta-data such as indexes, symbol
   tables, etc.

The Core Archive File Format is purposefully meant to be embraced and
extended while the primary feature of storing blobs of data with their
most crucial metadata, their "file-name", can be written by trivial
shell scripts and of course other languages.

## The Core Archive Specification

A "car" file is a sequence of "members" (where usually but not always
a member represents a file).

Each member has a human readable header followed by an optional binary
blob of data (the size of the blob comes from the "size:" key/value
pair in the header, see below).

The format is simple enough that it's sometimes legal to simply "cat"
together "car" files to create an archive that combines the
content. [Since this may violate some constraints, particularly around
alignment, it is not recommended and there will be a command line tool
to do this properly.]

Unlike "ar" files, "car" files do not have a magic number.

### Member Header Format

The header is a series of utf-8 encoded lines (though each line ends
in U+0000 instead of U+000A, i.e., these are actually null terminated
strings).

The header ends when a blank line is encountered (which practically
means that there will be at least two U+0000 in a row since a header
must have at least one initial line to hold the size: field).

Additional bytes containing U+0000 are some times appended to the
header as padding bytes to reach the alignment for the begining of the
data when the alignment is not 1).

While there is no mechanism to force the alignment of a header, if
every header specifies page boundaries for the data then all of the
headers will be aligned to page bounaries as well.

(For comparison, the "ar" format aligns headers and data to 16bit
boundaries which seems pretty useless since the natural alignment for
64bit machines would be 8 byte boundaries).

Each line of a member header is a single keys/value pair in the
following format:

<key in utf-8>: <value in utf-8>U+0000

The space (U+0020) following the ":" (U+003A) is *required* and only
one space is *allowed* after the ":".

Keys are arbitrary sequences of unicode utf-8 characters that don't
contain U+0000, U+0020, or U+003A). All of the "standard" keys (i.e.,
defined in this document) are 7-bit ASCII printable characters (a
subset of unicode utf-8).

Values are either base-10 integers (possibly with a leading "-") or
are meant to be treated as utf-8 strings (for example to hold the file
name metadata). U+0000 is the only unicode character not allowed in a
value string.

When parsing values to integers, readers should handle all numbers
representable in a 64bit 2's complement signed number (so up to 2^63
and down to -2^63). "-0" should always be considred to just be zero
(though implementations should *never* emit "-0" on purpose unless
they are simply preserving an incoming header).

The only key that *must* be present in the header is "size:" even
though a size zero is legal if there are no data contents for this
member. Even when there is a zero length data component, a header must
still be right padded to the alignment from the header using (U+0000)
after the blank line that is required to indicate the end of the
header.

Here is a nearly full set of well-known keys with some sample values:

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

(Additional well know keys not shown are for-file-name:,
is-binary-metadata, and external-file-name:)

This isn't a fully valid header because it *doesn't end in a blank
line* but otherwise I hope this should give a clear sense of how
things are represented.

TODO(jawilson): file ACLs and other file metadata? Data from MacOS or
Windows?

Without specifying a data-compression-algorithm, size: and data-size:
must be exactly the same (and therefore the data-size *should* be left
out). "size:" refers to the size of the data payload in the archive
(important for finding the next member of the archive) and
"data-size:" refers to the size of the data as would be present if
the uncompressed contents were written to disk.

The data-hash should always be computed before compression (that way
we can tell if the data compression algorithm actually preserved the
underlying data or not).

Keys that begin with "x-" are meant to be used for small amounts of
non-standard metadata that are not described in this document. Tools
should leave this metadata alone unless the user requests they be
removed.

It is sometimes legal to repeat keys with different values. For the
standard keys defined above, the first key value pair should be used
and for those keys, other key value pairs with the same key can be
dropped or ignored by tools.

It is *recommended* that alignment *not* be set for compressed members
(or simply specify the default "align: 1\n").

Current implementations can expect all "lines" in the header to be <=
4095 bytes long though most are expected to be much shorter. The core
archive file format allows arbitrary binary metadata to be associated
with a file for cases where this limit is unsuitable. (See below.)

Implementations should be able to give up if the size of any header
seems overly large since the file is possibly corrupt or an
implementation was given a file in the wrong format. Allowing the user
to override whatever limit is in place for sanity reason is probably
wise. This limit has not yet been decided but I'm unlikely to decide
anything less than 64KB (or obviously if the header extends past the
end of the "car" file itself).

### Member Data Format

As noted, the member data can be aligned (based on the offset with-in
the archive file modulo the alignment). Additionally, the member data
should be padded right padded to the appropriate alignment as well by
using trailing zero bytes to match the alignment. The rationale is to
allow "car" files to exist that have all data aligned to architecture
specific page boundaries for reading the data without potentially
reading data for a header (that is probably not related to that entry
itself, possbily a data leak!)

Otherwise, the member data is just the raw data bytes in a file, a
compressed version of the raw data using a compression algorithm, or
some application specific data (for example, indexes of various sorts
meant to make finding a specified part of the "car" file much easier).

If compression is used, we recommend using application/gzip for
general purpose core archive files since that is very widely available
and will be supported by most extraction tools. (And of course a
command line utility will be available to uncompress these for the
tools that can't even handle that).

Compression is *not* used by default (one can always compress the
entire "car" archive with a compression algorithm of ones choice,
though not all tools would understand that without first decompressing
the "car" file especially if the compression algorithm isn't
application/gzip).

## Indexes, Symbol Tables, Etc.

It may be desireable to store additional metadata in an efficient
binary format that either describes the entire archive file itself or
is limited to specific files in the archive.

When a header is describing such a metadata blob, it can omit the
"file-name:" key (unless there is a well-defined name for it already)
and add the line "is-binary-metadata: true" so other tools know this
data doesn't necessarily need to be extracted when the archive is
"extracted" to the file system.

When this binary metadata is about a particular file (or files) in the
archive, the key "for-file-name:" can be used (multiple times, the
only standard header that works this way).

The size: attribute must still be set as always and the mime-type:
should also be set to something appropriate.

If "for-file-name:" key is not used and the appropriate mime-type is
insufficient to locate the desired additional metadata, then the
standard key "identifier:" can be used.

We also recommend using one or more additional key/value pairs so that
a consumer of this index can determine if it is up to date or not (the
exact recommendation is TBD) since tools that manipulate a "car" file
could add new files or delete other files without updating this
application specific metadata.

(TBD: maybe a standard key to specify what program can be run on the
archive to regenerate said metadata?).

## Multiple Entries with the same file-name and Version Numbers

The car format allows multiple members with the same file-name as long
as they have different version numbers. By default, the highest
versioned member should be "returned" when requesting a member by name
without an explicit version number. Naturally some archives will be
produced that have the same file-name but no version information or
the same version number. In this case the library should provide a way
to get the first occurence and ignore the other members (though it can
indicate a warning condition).

## Lite Archives

"ar" provides a format whereby only metadata is supplied, and the data
contents are empty and are expected to be found in the file-system
(somehow).

For core archives, one merely needs to set size: to 0 and use
external-file-name: for a member instead of file-name:. Simple, right?

# Discussion

core archive files will likely have a slightly larger foot-print than
"ar" files because of the uncompressed fully human readable member
headers ("ar" headers are fixed size and because of this they have
caused great confusion regarding long file names and unicode
characters in file names).

The only unicode code-point that isn't allowed in file names is
U+0000.

Consideration was made for using LEB128, especially for size headers,
rather than base 10 human readable strings. Since an LEB128 encoded
numbers may contain either U+0000 or U+000A, this would require more
smarts and the small savings in space would mean every client needs to
be more aware of what they are processing. This doesn't seem like a
good trade-off. LEB128 remains a workable candidate for application
defined binary encoded metadata however.

I considered a different format for values, namely, C/Java/Javascript
style strings using "\" as an escape sequence (and of course
supporting \uXXXX to retain full unicode support). That would have
required more logic in all the libraries that process these values.

I'm still considering a required archive header (which could then also
contain a magic number) and potentially list features that a tool must
support to manipulate a particular "car" file.

# Deterministic Builds

If a core archive file is the output of a build step and the input to
another build step then it may be desireable to omit lots of useful
but irrelevant metadata and instead rely on the "data-hash-algorithm"
and the "data-hash" fields instead of say the posix information,
especially "posix-modification-time" and the user/group information.

# Implementations

The first generator of a "car" file will likely be a bash script that
uses expected binaries like echo on a unix/linux implementation to
generate a legal "car" file.

We'll update the this list of implementations once more are ready for
prime-time.

## Caching and Indexing

Libraries that operate on "car" files can provide "random access" to
items in the "car" file once they know what the offsets to members are
known. It's recommended that libraries remember what-ever they have
learned about a "car" file to speed up future access and of course
reclaim space once the "car" file is closed.

Since "car" files are meant to be written once, and accessed multiple
times, a tool could place an index as the first member of the the
"car" file to vastly speed up random access to individual files.

# Conclusion

The Core Archive File Format is a proposal for a "universal" and
extensible archive format that is extremely easy to produce and almost
as easy to read. Alignment and padding makes it suitable for use with
memory mapped files.

