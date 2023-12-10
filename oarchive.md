oarchive(1) -- archive tool for `.oar` archive format
=====================================================

## SYNOPSIS

`oarchive` COMMAND <FLAG>* <ARGS>*

## DESCRIPTION

`oarchive` can create, list, extract, append to, or join archives in
the omni archive format (`.oar`). Omni archive files are a
simplification of all existing file formats while having desirable
properties such as human readable (and extensible) metadata, and
un-restricted UTF-8 filenames.

## COMMAND

  * **create**, create an archive from the file patterns listed as
    ARGS. If --output-file is specified, the archive is written to
    that file otherwise the archive is written to stdout.

  * **list**, list all of the members in the archive specified by the
    --input-file argument. <ARGS> are treated as wild cards that
    restrict which files are listed so that list can act as a preview
    for extract.

  * **extract**, extracts all matching members to the current
     directory or to --output-dir. The input archive is either read
     from stdin or is specified by the --input-file flag. By default
     all members match otherwise the <ARGS> are treated as wild-card
     specifications and the member filename must match at least one of
     the wild-cards.

  * **append**, appends additional files to an archive from `stdin` or
      `--input-file` flag value.

  * **join**, appends two or more archives specified via
      ARGS. Currently this is not different from using `cat(1)` to append
      the raw bytes of the archives though in the future, this tool
      will recreate any standard indexes that are later added to the
      spec to allow efficient extraction/reading of individual members. 

## SEE ALSO

https://github.com/jasonaaronwilson/omni-archive

## AUTHOR

Jason A. Wilson
