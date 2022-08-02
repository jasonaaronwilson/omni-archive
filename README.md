# Core Archive File Format (car file)

The "car" file format (MIME type application/x-core-archive), is a
redesign of the (various) "ar" formats and "car" files use semi human
readable variable length member headers rather than a format partially
designed *just* to be "easy" for C to handle (i.e, fixed length
file-name fields or big or little endian fixed length strings).

Member headers are used to store metadata about the file contents such
as the size (required), filename, and other metadata normally
associated with an individual file in a file system and of course the
raw data content of each member.

The core archive file format supports:

1. aligning raw data (for example to 64-bits or page sizes)
2. checksums on the raw data (to detect corruption)
3. data compression of the raw data
4. unlimited length utf-8 file-names
5. posix file metadata
6. simple application specific metadata extensions
7. application specific binary meta-data such as indexes, symbol
   tables, etc.

The Core Archive File Format is purposefully meant to be embraced and
extended while the primary feature of storing blobs of data with their
"file-name" is relatively simple.

## The Core Archive Specification

A "car" file is a sequence of variable length "member headers" (sorted
by file-name or member name) followed by the raw data for members
(that have raw data). Members usually represent files though sometimes
they represent pure meta-data stored either in the header or as
application specific meta-data stored as raw data.

Unlike "ar" files, "car" files do not have a magic number.

Here is a graphic represention:

```
[file 1 header]
[file 2 header size:0]
[file 3 header]
[0, i.e., and empty header]
[zero byte filled padding]
[file 1 raw data]
[zero byte filled padding]
[file 3 raw data]
[zero byte filled padding]
```

### Member Header Format

A header is a series of key/value utf-8 encoded stings. In order to
support any legal utf-8 string as a value, these string are prefixed
with an ULEB128 encoded byte length. Each header ends with an empty
key/value string, i.e., a single byte value of 0. The header area ends
when the end of file is reached or else the offset of the earliest raw
data blob is reached.

Each line of a member header string is a single keys/value pair in the
following format (where things in {} are placeholders).

```
{key in utf-8}:{value in utf-8}
```

Keys are arbitrary sequences of unicode utf-8 code-points that don't
contain U+003A). All of the "standard" keys (i.e., defined in this
document) are 7-bit ASCII printable characters (a subset of unicode
utf-8).

Values are either base-16 encoded integers (using lower-case digits
"a" through "f" possibly with a leading "-" and then possibly with one
or more leading "0" digits) or are meant to be treated as utf-8
strings (for example to hold the file-name).

When parsing values to integers, readers should handle at least all
numbers representable in a 64bit 2's complement signed number (so up
to 2^63 and down to -2^63). "-0" should always be considered to just
be zero (though implementations should *never* emit "-0" on purpose
unless they are simply preserving an incoming header).

Each member header must contain the size: key/value pair as well as
one of the following key/value pairs: file-name:, metadata-name:, or
external-file-name:. size: is requires even when it is zero.

Here is a nearly full set of well-known keys with some sample values:

```
align:1
data-compression-algorithm:application/gzip
data-hash-algorithm:SHA-256
data-hash:784f6696040e7a4eb1465dacfaf421a526d2dd226601c0de59d7a1b711d17b99
data-size:302f
file-name:foo.txt
file-version:17
mime-type:text/plain
posix-file-mode:-rw-r--r--
posix-group-number:100
posix-group-name:jawilson
posix-modification-time-seconds:fffff
posix-modification-time-nanos:78ef
posix-owner-number:100
posix-owner-name:jawilson
size:18d6
start:f000
```

(Additional well known keys not shown are metadata-name:, for-file-name:, and
external-file-name: which would have been illegal to set in this
example)

This isn't a fully valid header because we aren't showing the encoded
string lengths of each string and strings don't actually end in a
newline but otherwise I hope this should give a clear sense of how
things are represented despite these small details. Headers are
obviously meant to be somewhat human readable.

TODO(jawilson): file ACLs and other file metadata? Data from MacOS or
Windows?

Keys that begin with "x-" are meant to be used for header inlined
non-standard metadata that are specific to certain applications. Tools
should preserve this metadata unless the user requests they be
removed.

It is illegal to repeat a key in a header. Instead, a format such as
x-my-key/0:, x-my-key/1, etc. can be used when that is truly needed.

### Member Data Format

Members are just raw bytes that appear anywhere after the header data
and don't necessarily need to be in the same order as the
headers. These are located using the "start" offset (relative to the
begining of the file). Technically when the same exact data contents
with the same alignment we could point at the same raw data from
different member headers, then the raw data could be emitted only once
but processing tools are allowed to duplicate these when say combining
archives and current libraries don't try to explicitly handle this
case.

As noted, the member data is sometimes aligned and tools must preserve
this alignment when joining together archives and members should be
zero padded according to the same alignment. The rationale is to allow
"car" files to exist that have all data aligned to either 64bit
boundaries or page boundaries for reading the data without potentially
reading data for a header.

Otherwise, the member data is just the raw data bytes in a file, a
compressed version of the raw data using a compression algorithm, or
some application specific data (for example, indexes of various sorts
meant to make finding a specified part of the "car" file much easier).

If compression is used, we recommend using application/gzip for
general purpose core archive files since that is very widely available
and will be supported by all but the simplest tools. (And of course a
command line utility will be available to rewrite an archive
completely uncompressed for the tools that can't even handle that).

Compression is *not* used by default (one can always compress the
entire "car" archive with a compression algorithm of ones choice,
though not all tools would understand that without first decompressing
the "car" file especially if the compression algorithm isn't
application/gzip).

## Indexes, Symbol Tables, Etc.

It may be desireable to store additional metadata in an efficient
binary format that either describes the entire archive file itself or
is limited to a specific file in the archive.

When a header is describing such a metadata blob, it should omit the
"file-name:" key/value and use "metadata-name:" instead (and possibly the
for-file-name: key). Since the value of the id may be used when the
user wants to extract these meta-data blobs to the file system for
examination, etc., they should probably kept human readable. The
inline meta-data for such blobs is by default written to
archive-metadata/{id} or archive-metadata/{for-file-name}/{id}).

When this binary metadata is about a particular file in the archive,
the key "for-file-name:" can be used. In this case the metadata-name: might be
though as a "type" of meta-data.

The size: attribute must still be set as always and the mime-type:
should also be set to something appropriate for clarity.

We also recommend using one or more additional key/value pairs so that
a consumer of this index can determine if it is up to date or not (the
exact recommendation is TBD) since tools that manipulate a "car" file
could add new files or delete other files without updating this
application specific metadata.

## Versions

The car format allows multiple members with the same file-name as long
as they *all* have version numbers and these are all distinct. By
default, the highest versioned member should be "returned" when
requesting a member by name without an explicit version number.

## Lite Archives

"ar" provides a format whereby only metadata is stored, and the data
contents are expected to be found in the file-system.

For core archives, one merely needs to set size: to 0 and use
external-file-name: for a member (instead of file-name:). In this case
the version: field must not be present.

# Standard Keys

Most keys are optional or only required when another field is set.

## align:

The alignment in hexidecimal (Y) as in 2^Y. Y=1 is the defalut
alignment and simply means byte aligned. Y=3 would mean 8 byte/64bit
alignment, and Y=c would mean 4096 byte alignment. Aligning on page
boundaries make core archive physically larger but makes memory
mapping individual raw data member easier.

## data-compression-algorithm: and data-size:

When either is present, both must be set. Additionally size: should be
present and > 0 (since compression is useless when the size is zero).

data-size: gives us the uncompressed length and data-size: must only
be set when data-compression-algorithm: is also set.

The most widely supported format is application/gzip and all but the
simplest libraries should support it.

## data-hash-algorithm: and data-hash:

When either is present, both must be set.

Many readers can ignore these when reading though command line tools
that unarhive should provid a command line option for checking these
after extraction (or can check them by default).

The data-hash: should always be computed before compression (that way
we can tell if the data compression algorithm actually preserved the
underlying data or not).

## file-name:, external-file-name:, metadata-name:, for-file-name:, and path-seperator:

Only one of file-name:, metadata-name:, and external-file-name: should be
set. When metadata-name: is set, the for-file-name: field can also be set.

file-name: is meant to specify an absolute or relative full file-path
and name using either the default path-seperator character "/" or
another path seperator character sequence such as "\" (for example on
windows).

## file-version:

A positive integer encoded in hexidecimal.

When multiple members with the same file-name are present, this serves
to differentiate them. File systems that support version numbers are
pretty rare though. Extraction tools may append "~N~" to the file-name
when extracting the members that aren't the highest version number
(where N is actual a base-10 number) though by default will only
extract the highest version of a file.

## mime-type:

This field is required when metadata-name: is used though encouraged
for other members too. For example, if a core-archive represents a
binary library file and has embedded data that can be accessed at
run-time, the mime-type: may be useful.

When used for indexes, tools may support mapping of a mime-type: to
another tool which can recompute binary metadata once the archive is
first created (this is similar to how "ar" may require running
"ranlib" on some systems).

## posix-file-mode:

The human readable posix file mode (not the octal based numbers).

## posix-group-name:

This is the group name of the file.

## posix-group-number:

This is a hexidecimal group id. We recommend using posix-group-name:
instead where possible.

## posix-modification-time-seconds: and posix-modification-time-nanos:

If posix-modification-time-nanos: is present then
posix-modification-time-seconds: must also be set even if zero.

The combination provides nanos from January 1, 1970 (time-zone
"Z"). The seconds may be a negative number though nanos are always
stored as a positive number.

## posix-owner-name:

This is the owner name of the file.

## posix-owner-number:

This is a hexidecimal owner id. We recommend using posix-owner-name:
instead where possible.

## size:

This is the hexidecimal size of the associated raw data for a member
and must be set for all members even if zero. Note that left padding
hexidecimal numbers with one or more ASCII "0" digits is often
employed to make writing a core achive file easier because we can't
choose offsets for the raw data associated with a memember until the
entire size of all the header files is not known since headers are
inherently variable length. A writer will typically fully encode all
of the headers and then either modify all of the sizes (and the start
offsets) or simply regenerate the headers completely now that these
can be determined.

The standard tool will likely make either two or three attempts at
generating headers. The first time it is assumes that everything is
lower than ffffffff (8 digits, 32 bits) and can obviously abort if
this is determined not to be true (or obvious from the start when the
sum of the data sizes are know to be greater than 2^32 or sufficiently
close to it assuming padding and an approximation of the header size
itself times the number of headers). The last attempt resorts to using
16 hexidecimal digits and would only fail if the resulting core
archive file is larger than 2^64 bytes.

## start:

This is the offset relative to the begining of the file stored as a
positive hexidecimal number. start: is required when size: > 0. See
size: to understand why these may often be encoded with left padded
"0" digits to simplify writers.

# Discussion

core archive files may have a slightly larger foot-print than "ar"
files because of the uncompressed semi human readable member headers
("ar" headers are fixed size and because of this they have caused
great confusion regarding long file names and unicode characters in
file names).

Placing headers at the beginning of a file and sorting them makes
generation more difficult but then allows a scan of only the begining
of the file to find where a particular member's raw data is (and a
binary search may be possible directly on the headers when an entire
core-archive is in memory (for example, memory-mapped or embedded in
an executable).

Consideration was made for using unsigned ULEB128 to encode number
fields inside of the header key/value strings but the saving would
probably be less than about 16 bytes per member (or 32 bytes per
member when the total core archive is larger than 2^32 bytes).

I considered a different format for values, namely, C/Java/Javascript
style strings using U+005C as an escape sequence (and of course
supporting \uXXXX to retain full unicode support). That would have
required more logic in all the libraries that process these values. I
also considered making header key/value strings actual unix style
lines (i.e., ending in U+000A) and then simply ending them with
U+0000. It turns out both values are valid as file-names (at least on
some systems) and hence the ULEB128 length prefix was ultimately
decided on to allow no limitations on file-names except being valid
utf-8.

# Deterministic Builds

If a core archive file is the output of a build step and the input to
another build step then it may be desireable to omit lots of useful
but irrelevant metadata and instead rely on the "data-hash-algorithm"
and the "data-hash" fields instead of say the posix information,
especially "posix-modification-time" and the user/group information.

# Implementations (command line tools and libraries)

We'll update the this list of implementations right here once they are
ready for prime-time.

# Conclusion

The Core Archive File Format is a proposal for a "universal" and
extensible archive format that is extremely easy to produce and
consume. Alignment and padding makes it suitable for use with memory
mapped files.
