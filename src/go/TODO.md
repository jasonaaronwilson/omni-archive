# core-archive-command.go

1. write out zero length files as long as they are actually files
2. add some simple tests (make test does the simplest test right now)
3. allow creating an archive of a directory, recursively, (though
   technically find plus xargs would probably suffice for most needs
   and I started with even a more basic need)
4. full archive extraction (via a refactor)
5. warn on potential over writing of a file that already exists
6. detect duplicate filenames and other potential errors. add the
   check-headers command

# core-archive-lib.go

1. obviously extract out the relevant parts of core-archive-command.go
2. start documenting the API
3. unit tests on reading and writing ULEB128?
4. more efficienty especially on copying from archive to an output
   file which we currently do one bye at a time
5. the key/value pairs order should be deterministic (so probably just
   sort on key names)
