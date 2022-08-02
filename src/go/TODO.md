# core-archive-command.go

1. write out zero length archives as long as they are actually files
2. add some simple tests
3. allow creating an archive of a directory, recursively, (though
   technically find plus xargs would probably suffice)
4. full archive extraction (via a refactor)
5. warn on potential over writing...

# core-archive-lib.go

1. obviously extract out the relevant parts of core-archive-command.go
2. start documenting the API
3. unit tests on reading and writing ULEB128?
4. more efficienty on copying from archive to an output file
5. the key/value pairs order should be deterministic (so probably just
   sort on key names)
