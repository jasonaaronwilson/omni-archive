package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// These are the only keys we can explicitly read and write though
// when appending archives, we preserve all of the key/value pairs
// even if they are not known.
const (
	FILE_NAME_KEY = "file-name:"
	SIZE_KEY      = "size:"
	START_KEY     = "start:"
)

// These are additional "known keys" from the spec
const (
	ALIGN_KEY                           = "align:"
	DATA_COMPRESSION_ALGORITHM_KEY      = "data-compression-algorithm:"
	DATA_HASH_ALGORITHM_KEY             = "data-hash-algorithm:"
	DATA_HASH_KEY                       = "data-hash:"
	DATA_SIZE_KEY                       = "data-size:"
	EXTERNAL_FILE_NAME_KEY              = "external-file-name:"
	FILE_VERSION_KEY                    = "file-version:"
	FOR_FILE_NAME_KEY                   = "for-file-name:"
	METADATA_NAME_KEY                   = "metadata-name:"
	MIME_VERSION_KEY                    = "mime-version:"
	POSIX_FILE_MODE_KEY                 = "posix-file-mode:"
	POSIX_GROUP_NAME_KEY                = "posix-group-name:"
	POSIX_GROUP_NUMBER_KEY              = "posix-group-number:"
	POSIX_MODIFICATION_TIME_NANOS_KEY   = "posix-modification-time-nanos:"
	POSIX_MODIFICATION_TIME_SECONDS_KEY = "posix-modification-time-seconds:"
	POSIX_OWNER_NAME_KEY                = "posix-owner-name:"
	POSIX_OWNER_NUMBER_KEY              = "posix-owner-number:"
)

const (
	USER_DEFINED_KEY_PREFIX = "x-"
)

//
// Read all headers and display the file names contained in a very
// succinct format.
//
func list_command(args []string) {
	for _, archive_name := range args {
		with_archive(
			archive_name,
			func(archive *os.File) {
				headers := read_headers(archive)
				for _, header := range headers {
					fmt.Println(header[FILE_NAME_KEY])
				}
			})
	}
}

func create_command(args []string) {
	archive_name := args[0]
	files := args[1:]

	headers := []map[string]string{}

	for _, member := range files {
		// TODO(jawilson): verbosity and write to standard error
		fmt.Println("Adding " + member)
		header := make(map[string]string)
		header[FILE_NAME_KEY] = member

		fi, err := os.Stat(member)
		if err != nil {
			panic(err)
		}

		if fi.IsDir() {
			panic("Can't handle directories yet")
		}

		header[SIZE_KEY] = fmt.Sprintf("%x", fi.Size())

		headers = append(headers, header)
	}

	write_archive(archive_name, headers)
}

func extract_files_command(args []string) {
	archive_name := args[0]
	files := args[1:]

	archive, err := os.Open(archive_name)
	if err != nil {
		panic(err)
	}

	headers := read_headers(archive)

	// Now extract each file

	// TODO(jawilson): organize headers into a map of headers off
	// the key FILE_NAME_KEY so this isn't O(N^2) where N is the
	// number of headers

	for _, filename := range files {
		header := find_header(headers, filename)
		if header == nil {
			panic("File not found in archive: " + filename)
		}
		write_from_file_offset(archive,
			filename,
			as_int64(header[START_KEY]),
			as_int64(header[SIZE_KEY]))
	}

	if err := archive.Close(); err != nil {
		panic(err)
	}
}

func with_archive(archive_name string, thunk func(*os.File)) {
	archive := open_archive(archive_name)
	thunk(archive)
	if err := archive.Close(); err != nil {
		panic(err)
	}
}

func open_archive(archive_name string) *os.File {
	archive, err := os.Open(archive_name)
	if err != nil {
		panic(err)
	}
	return archive
}

func find_header(headers []map[string]string, filename string) map[string]string {
	for _, header := range headers {
		if header[FILE_NAME_KEY] == filename {
			return header
		}
	}
	return nil
}

//
// Assign START_KEY values to all members with non-zero size.
//
// For every member with non-zero size, we need to set a value for
// "start:" such that their raw contents don't overlap (or overlap
// with a header) and of course to do this without any wasted space
// (modulo alignment which is currently not supported).
//
// Of course we don't really know where the first file should start
// without computing the size of all of the headers and that presents
// a problem since the size is technically a variable width
// quantity. To work around this, we first compute the size of all of
// the headers using a fixed width size string (00000000) and then as
// long as the actual writing also left pads the hexidecimal size to
// the same number of digits then we know how big the header is. We
// also need to add one since we always add a blank "header" according
// to the specification (this makes is much easier to determine where
// the last header is).
//
// TODO(jawilson): handle alignment.
//
func layout_archive(headers []map[string]string) {
	header_size := 0
	for _, member := range headers {
		if as_int64(member[SIZE_KEY]) > 0 {
			member[START_KEY] = "00000000"
			header_size += len(header_to_bytes(member))
		}
	}
	// We always write an extra 0 byte after the headers (which
	// reads as an empty header) and this tells a reader where the
	// end of the headers is.
	header_size += 1
	start := int64(header_size)
	for _, member := range headers {
		if start > (1 << 31) {
			panic("archive is currently to too large")
		}
		if as_int64(member[SIZE_KEY]) > 0 {
			member[START_KEY] = fmt.Sprintf("%08x", start)
			start += as_int64(member[SIZE_KEY])
		}
	}
}

// Convert a possibly zero prefixed hexidecimal number to and int64 or
// panic.
func as_int64(value string) int64 {
	num, err := strconv.ParseInt(value, 16, 64)
	if err != nil {
		panic(err)
	}
	return num
}

//
// Given an output archive filename and a set of headers (which must
// include the sizes of all written elements), writes an archive file
// by first laying out the archive, then writing all of the headers,
// and then finally writing all of the files contents (currently all
// read from disk which wont' work nicely when trying to append
// archives).
//
func write_archive(archive_name string, headers []map[string]string) {
	/* First we need to figure out where everything goes */
	layout_archive(headers)

	/* Open the output file */
	fo, err := os.Create(archive_name)
	if err != nil {
		panic(err)
	}

	/* First write all of the headers */
	for _, member := range headers {
		if _, err := fo.Write(header_to_bytes(member)); err != nil {
			panic(err)
		}
	}

	/* Write and empty header / zero byte to signal the end of headers. */
	if _, err := fo.Write([]byte{0}); err != nil {
		panic(err)
	}

	/* Now write all of the raw data contents */
	for _, member := range headers {
		if as_int64(member[SIZE_KEY]) > 0 {
			write_file_contents(fo, member[FILE_NAME_KEY])
		}
	}

	/* Close the output file */
	if err := fo.Close(); err != nil {
		panic(err)
	}
}

// Opens the file "filename" and simply appends all of its bytes to
// "output".
func write_file_contents(output *os.File, filename string) {
	fi, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 4096)
	for {
		n, err := fi.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		if _, err := output.Write(buf[:n]); err != nil {
			panic(err)
		}
	}
	if err := fi.Close(); err != nil {
		panic(err)
	}
}

//
// This is a debugging routine that creates a textual version of a
// header to show a user.
//
func header_to_string(header map[string]string) string {
	result := ""
	for key, value := range header {
		result += key
		result += value
		result += "\n"
	}
	return result
}

//
// This actually converts a header to its representation of a header
// which is a series of ULEB128 length prefixed utf-8 strings that
// happen to all start with with the regexp ".*:".
//
// A header always ends with a byte of zero which is an empty string
// and this routine always emits such an empty line.
//
func header_to_bytes(header map[string]string) []byte {
	result := []byte{}
	for key, value := range header {
		result = append(result, key_value_pair_to_bytes(key, value)...)
	}
	result = append(result, 0)
	return result
}

// TODO(jawilson): this should create a single byte of zero if both
// key and value are empty.
func key_value_pair_to_bytes(key string, value string) []byte {
	result := []byte{}
	result = append(result, []byte(key)...)
	result = append(result, []byte(value)...)
	result = append(uleb128(int64(len(result))), result...)
	return result
}

// Encode an int64 as bytes according to the common definition (see
// wikipedia) of an unsigned LEB128 number. This should result in
// between 1 and 10 bytes.
func uleb128(number int64) []byte {
	result := []byte{}
	for {
		var b byte = byte(number & 0x7f)
		number = number >> 7
		more := number > 0
		if more {
			b |= (1 << 7)
		}
		result = append(result, b)
		if more {
			continue
		}
		return result
	}
}

// Read sequences of ULEB128 prefixed strings into a sequence of
// "header" objects (i.e. map[string]string). Each header stops when
// we encounter a single terminating zero byte (aka, empty "line")
// that isn't itself the terminator for a header. While all header
// sequences end in 0x0, 0x0, this may not be the first such
// appearance of two zeros in a row (for example, a degenerate
// filename with two U+0000 characters in a row).
func read_headers(archive *os.File) []map[string]string {
	result := []map[string]string{}
	// end := int64(^uint64(0) >> 1)
	for {
		header := read_header(archive)
		if len(header) == 0 {
			break
		}
		fmt.Println(header_to_string(header))
		result = append(result, header)
	}
	return result
}

// Read a sequence of ULEB128 prefixed strings until we encounter an
// empty string. Convert all non-empty strings into a
// map[string]string where keys are all unicode characters preceding
// and including the first ":" and values are the rest of the string.
// This requires that the contents of a string be legal UTF-8, that
// there exists at least one ":" in each non empty line.
func read_header(archive *os.File) map[string]string {
	result := make(map[string]string)
	for {
		str := read_uleb128_prefixed_string(archive)
		if len(str) == 0 {
			break
		}
		key_end := strings.Index(str, ":") + 1
		result[str[0:key_end]] = str[key_end:]
	}
	return result
}

// This low-level routine reads an unsigned LEB128 from the current
// file and then reads that many bytes and converts those bytes to a
// proper legal unicode string.
func read_uleb128_prefixed_string(archive *os.File) string {
	str_length := read_uleb128(archive)
	str_bytes := make([]byte, str_length)
	n, err := archive.Read(str_bytes)
	if int64(n) != str_length {
		panic("Expected to read all the bytes requested")
	}
	if err != nil {
		panic(err)
	}
	return string(str_bytes)
}

// This low-level routine reads an unsigned LEB128 from the current
// file and returns it as an int64. Since LEB128 is only used to
// encode header strings, any value larger than about 4096 is probably
// fishy (hence int64 being a total over-kill despite "128" in the
// name.
func read_uleb128(archive *os.File) int64 {
	result := int64(0)
	shift := 0
	for {
		if shift >= 32 {
			panic("Encountered a ridicuously long ULEB128 string length")
		}
		b := read_byte(archive)
		result |= int64((b & 0x7f)) << shift
		if b&(1<<7) == 0 {
			break
		}
		shift += 7
	}
	// fmt.Printf("uleb128 is %d", result)
	return result
}

// Reads a single byte that must be present and panic if any errors
// occur
func read_byte(archive *os.File) byte {
	barray := make([]byte, 1)
	n, err := archive.Read(barray)
	if n != 1 {
		panic("Expected to read one byte")
	}
	if err != nil {
		panic(err)
	}
	return barray[0]
}

// Reads a single byte and panics if any errors occur
func write_byte(archive *os.File, b byte) {
	barray := []byte{b}
	n, err := archive.Write(barray)
	if n != 1 {
		panic("Expected to read one byte")
	}
	if err != nil {
		panic("Expected to write one byte")
	}
}

// Attempts to materialize in the filesystem as "filename" the bytes
// from "start" to "start+size" from an archive.
//
// TODO(jawilson): various posix information that should be preserved
// as well.
//
// TODO(jawilson): read and write in larger chunks than one byte!
func write_from_file_offset(input *os.File, filename string, start int64, size int64) {
	offset, err := input.Seek(start, 0)
	if offset != start {
		panic("failed to seek to correct position")
	}
	if err != nil {
		panic(err)
	}

	// open output file
	create_parent_directories(filename)
	output, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	for i := int64(0); i < size; i++ {
		write_byte(output, read_byte(input))
	}

	if err := output.Close(); err != nil {
		panic(err)
	}
}

// In order to write this file-name, ensure that all of its parent
// directories exist.
//
// TODO(jawilson): do we need to get posix info from the archive itself?
//
// TODO(jawilson): cache directories we know exist to avoid repeated
// calls to os.Stat which could be slow
func create_parent_directories(filename string) {
	dir_path := filepath.Dir(filename)
	if _, err := os.Stat(dir_path); os.IsNotExist(err) {
		// TODO(jawilson): what should be mode really be?
		err := os.MkdirAll(dir_path, 0750)
		if err != nil {
			panic(err)
		}
	}
}

// Output the usage for this tool.
func usage() {
	fmt.Println(`Usage:    
core-archive create {core-archive-filename} [filenames...]
core-archive extract-all {core-archive-filename}
core-archive extract-files {core-archive-filename} [filenames...]
core-archive append [archive 0] [archive 1] ...
core-archive list [archive 0] [archive 1] ...
core-archive headers [archive 0] [archive 1] ...
core-archive remove [archive 0] [filenames...]
core-archive update [filenames...]
core-archive --usage
core-archive --version`)
}

// Obviously the entry point to this tool.
func main() {
	if len(os.Args) <= 1 {
		usage()
		return
	}
	command := os.Args[1]
	command_args := os.Args[2:]
	switch command {
	case "create":
		create_command(command_args)
	case "extract-files":
		extract_files_command(command_args)
	case "list":
		list_command(command_args)
	default:
		usage()
	}
}
