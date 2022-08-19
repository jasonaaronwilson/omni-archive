# Core Archive File Format (car file)

The "car" file format (MIME type application/x-core-archive), is a
redesign of the "ar" format(s). "car" files use semi human readable
*variable* length member headers rather than a fixed length or an
endian dependent format.

Member headers are used to store metadata about the file contents such
as the size (required), filename, and additional metadata associated
with an individual file in a file system and of course the *raw (byte)
data* content of each member.

The core archive file format supports:

1. aligning raw data (for example to 64-bits or even machine page sizes)
2. checksums on the raw data (to detect corruption)
3. data compression of the raw data
4. unlimited length/content utf-8 file-names
5. posix file metadata
6. simple application specific metadata extensions
7. application specific binary meta-data such as indexes, symbol
   tables, etc.

The Core Archive File Format is purposefully meant to be embraced and
extended while the primary feature of storing blobs of data with their
"file-name" is still very simple.

## The Core Archive Specification

A "car" file is a sequence of variable length "member headers" (TODO:
sorted by file-name or member name?) followed by the raw data for
members that have raw data. Members usually represent files though
sometimes they represent pure meta-data stored either in the header or
as application specific meta-data stored as raw data.

Unlike "ar" files, "car" files do not have a magic
number. (TODO(jawilson): why not just define a magic number and
include a version scheme?)

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
with an ULEB128 encoded *byte* length (though most implementations
will complain about anything close to 2^63 bytes or even 2^31 bytes
since these are not where the bulk of the data is actually stored).

Each header ends with an empty key/value string, i.e., a single byte
value of 0.

The end of the entire header area ends when there is a header without
any elements (so practically speaking, two zero bytes in a row though
two zero bytes in a row are not sufficient to scan for).

Each member header string is a single keys/value pair in the following
format (where things in {} are placeholders).

```
{key in utf-8}:{value in utf-8}
```

Keys are arbitrary sequences of unicode utf-8 encoded code-points that
don't contain U+003A. All of the "standard" keys (i.e., defined in
this document) are 7-bit ASCII printable characters (a subset of
unicode utf-8).

Values are utf8 strings, i.e., everything after the ":". In practice,
when encoding "integers" for values of well defined keys, we use
base-16 encoded integers (using the "digits" drawn from the ASCII
characters 0123456789abcdef) possibly with a leading "-" and very
likely to contain left padded zeros (after the initial "-" if
present).

When parsing values to integers, readers should handle at least all
numbers representable in a 64bit 2's complement signed number (so up
to 2^63 and down to -2^63). "-0" should always be considered to just
be zero (though implementations should *never* emit "-0" on purpose
unless they are simply preserving an incoming header).

Each member header must contain the size: key/value pair as well as
one of the following key/value pairs: file-name:, metadata-name:, or
external-file-name:. size: is *required* even when it is zero.

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
string lengths and strings don't actually end in a newline to make
them pretty but otherwise I hope this should give a clear sense of how
things are represented despite these small details. Even in the full
binary format, headers are somewhat human readable.

Keys that begin with "x-" are meant to be used for header inlined
non-standard metadata that are specific to certain applications. Tools
should preserve this metadata unless the user requests they be
removed.

It is illegal to repeat a key in a header. Instead, use this format:
```
   x-my-key/0:
   x-my-key/1:
   x-my-key/{XYZ}:
```

where {XYZ} is a hexidecimal number though negative signs and left "0"
padding is *not* allowed.

### Member Data Format

The raw data for a member is just bytes that appear anywhere after the
header data and don't necessarily need to be in the same order as the
headers. These are located using the "start" offset (relative to the
begining of the file).

[When the same exact data contents and alignment occur, tools are
encouraged to "point" at the same raw data from different member
headers meaning this raw data can be emitted only once, however,
processing tools are allowed to duplicate these (perhaps making the
archive much bigger) when say combining archives.]

As noted, the member data is sometimes aligned and tools must preserve
this alignment when joining together archives and members should be
zero padded according to the same alignment. The rationale is to allow
"car" files to exist that have all data aligned to either 64bit
boundaries or even page boundaries such that a subset of an entire
"car" file can be memory mapped and not see or especially write data
that doesn't pertain to that "member" of a "car" file.

The raw member data is always either the raw data bytes in a file, a
compressed version of the raw bytes in a file using a compression
algorithm, or some application specific data (for example, indexes of
various sorts meant to make finding a specified part of the "car" file
much easier).

If compression is used, we recommend using application/gzip for
general purpose core archive files since that is very widely available
and will be supported by all but the simplest tools. (And of course a
command line utility will be available to rewrite an archive
completely uncompressed for the tools that can't even handle that).

Compression is *not* used by default (one can always compress the
entire "car" archive with a compression algorithm of one's choice,
though not all tools would understand that without first decompressing
the "car" file especially if the compression algorithm isn't
application/gzip).

## Indexes, Symbol Tables, Etc.

It may be desireable to store additional metadata in an efficient
binary format that either refers to a single file in an archive or the
entire archive itself!

When describing meta-data for a particular file, implementations
*must* use the meta-data-for-file-name:{file-name} and must not
include a file-name: (or meta-data: key).

When describing meta-data for an entire archive, the header
"meta-data:{name}" should be used. {name} should be "pathlike" since a
user may want to see this meta-data in a seperate file after
extraction.

When some binary metadata is about a particular file in the archive,
the key "for-file-name:" can be used (and

The size: attribute must still be set as always and the mime-type: is
super highly recommended for clarity.

We also recommend using one or more additional key/value pairs so that
a consumer of this index can determine if it is up to date or not (the
exact recommendation is TBD) since tools that manipulate a "car" file
could add new files or delete other files without updating this
application specific metadata.

## Versions

The car format allows multiple members with the same file-name as long
as they *all* have version numbers and these are *all* distinct. By
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

The alignment in hexidecimal (Z) as in 2^Z. Y=1 is the defalut
alignment and simply means byte aligned. Z=3 would mean 8 byte/64bit
alignment, and Z=c would mean 4096 byte alignment. Aligning on page
boundaries make core archive physically larger but makes memory
mapping individual raw data member easier especially when readers want
to prevent one reader from seeing something another reader can't see.

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

This field is required when metadata-name: is used though fully
encouraged for other members too. (If a core-archive represents a
binary library file and has embedded data that can be accessed at
run-time, the mime-type: may be highly useful, for example as part of
an HTTP response.)

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
instead where possible. When it's not possible to encode the full
resolution of a group number, tools should either panic or warn about
this.

## posix-modification-time-seconds: and posix-modification-time-nanos:

If posix-modification-time-nanos: is present then
posix-modification-time-seconds: must also be set even if zero.

The combination provides nano second resolution from January 1, 1970
(time-zone "Z"). The seconds may be a negative number though nanos are
always stored as a positive number.

## posix-owner-name:

This is the owner name of the file.

## posix-owner-number:

This is a hexidecimal owner id. We recommend using posix-owner-name:
instead where possible. When it's not possible to encode the full
resolution of a large group number, tools should either panic or warn
about this.

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

This is the offset relative to *the begining of the file* stored as a
positive hexidecimal number. start: is required when size: > 0. See
size: to understand why these may often be encoded with left padded
"0" digits to simplify writers.

# Discussion

core archive files may have a slightly larger foot-print than "ar"
files because of the uncompressed semi human readable member headers
("ar" headers are fixed size and because of this they have caused
great confusion regarding long file names and unicode characters in
file names).

Placing headers at the beginning of a file (and sorting them?) makes
generation more difficult but then allows a scan of only the begining
of the file to find where a particular member's raw data is (and a
binary search may be possible directly on the headers when an entire
core-archive is in memory and sorted (for example, memory-mapped or
embedded in an executable).

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
U+0000. It turns out both U+0000 and U+000A are sometimes valid as
file-names and hence the ULEB128 length prefix was ultimately decided
on to allow no limitations on file-names except being valid utf-8.

## Deterministic Builds

If a core archive file is the output of a build step and the input to
another build step then it may be desireable to omit lots of useful
but irrelevant metadata and instead rely on the "data-hash-algorithm"
and the "data-hash" fields instead of say the posix information,
especially "posix-modification-time" (and the user/group information
if you want to share across builds).

# Implementations (command line tools and libraries)

We'll update the this list of implementations right here once they are
ready for consideration or for extensive usage.

1. src/go/core-archive-command.go

This is getting near complete as a useful create/append/extract tool
(though doesn't support output alignment and POSIX info yet amoung
other things yet).

Since I'm the current author, this code will eventually be both a
library and a command line tool.

# Conclusion

The Core Archive File Format is a proposal for a "universal" and
extensible archive format that is extremely easy to produce and
consume. Alignment (and therefore padding) makes it suitable for use
with memory mapped files with differing permissions.

# TODO(jawilson)

We should store directory names so that we make get the exact group,
owner, and other metadata upon extraction.



TODO(jawilson): file ACLs and other file metadata? Data from MacOS or
Windows?

