# Omni Archive Format Extensibility and "Container" support

Omni archives are meant to be forward and backwards compatible which
is why they have variable headers that support UTF-8 key/value pairs.

Inspired by MIME types, we also use the "x-" prefix to denote a
metadata extension and tools must preserve these key/value pair lines
the same way as they must preserve standardize keys they don't know
about.

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


[^1] For example, if you were designing a file format for a word
processor, you might store the document text as one "logical"
file-name (maybe in XML?) and then every image in the document could
be stored as other "logical" file-names (presumably PNGs, JPGs, GIFs,
etc.) Maybe you really wanted each "chapter" to be it's own logical
file, you can do that too! Another example, if you wanted to create
something like a "web archive", perhaps each HTML, JS, and image files
could have a property like x-source-url to keep track of where these
were obtained.

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

