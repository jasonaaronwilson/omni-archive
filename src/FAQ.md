# Frequently Asked Questions

1. Should I use a core-archive tool like core-archive-command instead
of tar, zip, etc. for my backups?

Right now, there are few compelling reason to use "core-archive"
rather than say "tar" or one of many zip tools for backups,
transmission, etc. especially with a standard "POSIX" file-systems.

It appears that GNU tar may currently have a limit of about 256
characters for a filename in an archive and it's not clear how well it
deals with other unusual filenames. When there is a widely available
tool that surpasses GNU tar in this regards (and tests that prove it),
we will update this information here.

In the future, I hope that versions of of core-archive are available
on systems like legacy MacOS and that core-archive will provide a much
better experience than a POSIX only tool for backups and transmission
but such tools still need to be rigorously tested.

2. Should I use a core-archive to represent my new data format "XYZ"?

If decent library support is available for your language, then YES!
(And if no, a few hundred lines at most and you can create such
library support.)

core-archive was created precisely with this scenario in mind. You
want to program as though your new data format is really lots of
"blobs" of different data that you can access "randomly" in your code
(much like individnual files). However, your user expects a single
output file because that's how all software works.




