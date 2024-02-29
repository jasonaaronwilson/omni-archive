#!/bin/bash

set -x
rm -rf test-out
mkdir -p test-out

FILES=`find . -type f`
echo $FILES

./oarchive create --ouput-file=test-out/test-archive.oar $FILES

# Gemini claimed that * on it's own will not match a directory but I
# think it does though maybe not into the files in such a directory.

### ./oarchive create --ouput-file=test-out/test-archive.oar *

# ./oarchive create * > test-out/test-archive.oar
# ./oarchive list --input-file=test-out/test-archive.oar
# cat test-out/test-archive.oar | ./oarchive list

# ./oarchive create * | ./oarchive list
# cd test-out
# ../oarchive list --input-file=test-archive.oar




