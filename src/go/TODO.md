# core-archive-command.go

1. write out zero length files as long as they are actually files
2. add some simple tests (make test does the simplest test right now)
3. warn on potential over writing of a file that already exists
4. detect duplicate filenames and other potential errors. add the
   check-headers command?
5. make sure setting verbosity doesn't make tests fail
6. MAYBE sort headers. the biggest reason to do this is
   reproducibility but we can potentially achieve this in other ways

DONE

Performance was terrible (actually, I only tested creating an archive
but there is no reason to expect that extraction wouldn't also be slow
now that we've sped up creation.) I thought it would be bad because we
are doing "byte" IO but it was more than 100X slower than tar which
did surprise me a bit. Using an 8K buffer reduces the time from
minutes to 0.336s (tar is at 0.252s so we are now maybe only 33%
slower than tar which is written in bare metal C and has more than 25X
the amount of SLOCs).

# core-archive-lib.go

1. obviously extract out the relevant parts of core-archive-command.go
2. start documenting the API
3. unit tests on reading and writing ULEB128?
4. more efficienty especially on copying from archive to an output
   file which we currently do one bye at a time
