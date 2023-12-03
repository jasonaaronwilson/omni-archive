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

