package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

// Application specific keys are prefixed with "x-".
const (
	USER_DEFINED_KEY_PREFIX = "x-"
)

var verbosity uint = 0

const (
	VERBOSITY_ERROR   = 0 // verbosity is unsigned so errors are always shown
	VERBOSITY_WARNING = 1
	VERBOSITY_INFO    = 2
)

// The copy_bytes_to_output achieves *massive* speedups by reading and
// writing in chunks instead of one byte at a time. Since we don't try
// to reuse the allocated buffer (and for other reasons), for now we
// are just keeping this at a reasonable size. For testing, I
// sometimes set this to 3 (it would actually be cool to set this from
// the command line).
const (
	BUFFER_SIZE = 8192
)

// This represents both IO sources and IO targets.
type IOInfo struct {
	// Only one of filename or file should be set
	filename string
	file     *os.File

	// A value of zero means do not perform a seek before reading
	seek_offset int64

	// For an input, this tells us how many bytes to expect. For
	// an output, it has no meaning and should be left zero
	size int64
}

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
					if has_key(header, FILE_NAME_KEY) {
						fmt.Println(header[FILE_NAME_KEY])
					}
				}
			})
	}
}

//
// Read all headers and then display them in a human readable format
//
func headers_command(args []string) {
	for _, archive_name := range args {
		with_archive(
			archive_name,
			func(archive *os.File) {
				headers := read_headers(archive)
				for _, header := range headers {
					fmt.Println(header_to_string(header))
				}
			})
	}
}

//
// This command appends one or more archives.
//
func append_command(args []string) {
	archive_name := args[0]
	archives := args[1:]

	headers := []map[string]string{}
	inputs := []IOInfo{}
	to_close := []*os.File{}

	for _, input_archive_name := range archives {
		archive, err := os.Open(input_archive_name)
		if err != nil {
			panic(err)
		}
		to_close = append(to_close, archive)
		more_headers := read_headers(archive)
		for _, header := range more_headers {
			// TODO(jawilson): we can have a header with zero size...
			// if has_key(header, FILE_NAME_KEY) {
			// }
			headers = append(headers, header)
			inputs = append(inputs,
				IOInfo{
					file:        archive,
					seek_offset: as_int64(header[START_KEY]),
					size:        as_int64(header[SIZE_KEY]),
				})
		}
	}

	write_archive(archive_name, headers, inputs)

	// Close all of the archives we've opened
	for _, openFile := range to_close {
		if err := openFile.Close(); err != nil {
			panic(err)
		}
	}
}

//
// This command creates an archive based on the command line
// arguments.
//
func create_command(args []string) {
	archive_name := args[0]
	files := args[1:]

	headers := []map[string]string{}
	inputs := []IOInfo{}

	for _, root := range files {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				panic(err)
			}
			if info.IsDir() {
				return nil
			}
			if verbosity >= VERBOSITY_INFO {
				fmt.Println("Adding " + path)
			}

			header := make(map[string]string)
			header[FILE_NAME_KEY] = make_path_relative_if_absolute(path)
			header[SIZE_KEY] = fmt.Sprintf("%x", info.Size())

			headers = append(headers, header)
			inputs = append(inputs,
				IOInfo{
					filename: path,
					size:     info.Size(),
				})
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	write_archive(archive_name, headers, inputs)
}

func make_path_relative_if_absolute(path string) string {
	if strings.HasPrefix(path, "/") {
		asRunes := []rune(path)
		if len(asRunes) == 1 {
			panic("Removed the only character '/' for a path")
		}
		return string(asRunes[1:])
	}
	return path
}

//
// This command allows the removal of some members from an archive
//
func remove_by_filename_command(args []string) {
	output_archive_name := args[0]
	input_archive_name := args[1]
	to_remove_names := args[2:]
	to_remove_map := make(map[string]bool)
	for _, name := range to_remove_names {
		to_remove_map[name] = true
	}

	headers := []map[string]string{}
	inputs := []IOInfo{}
	to_close := []*os.File{}

	archive, err := os.Open(input_archive_name)
	if err != nil {
		panic(err)
	}
	to_close = append(to_close, archive)
	more_headers := read_headers(archive)
	for _, header := range more_headers {
		if to_remove_map[header[FILE_NAME_KEY]] {
			continue
		}
		headers = append(headers, header)
		inputs = append(inputs,
			IOInfo{
				file:        archive,
				seek_offset: as_int64(header[START_KEY]),
				size:        as_int64(header[SIZE_KEY]),
			})
	}

	write_archive(output_archive_name, headers, inputs)

	// Close all of the archives we've opened
	for _, openFile := range to_close {
		if err := openFile.Close(); err != nil {
			panic(err)
		}
	}
}

// Only extract *files* explicitly requested on the command
// line. (Since shells and POSIX style filenames (unless explicitly
// ending in say "/") it's hard to tell directories from files to
// infer intent).
//
// TODO(jawilson): it looks like this can use
// extract_files_by_predicate shortly.
func extract_by_file_name_command(args []string) {
	archive_name := args[0]
	files := args[1:]

	with_archive(archive_name,
		func(archive *os.File) {

			headers := read_headers(archive)

			// Now extract each file

			// TODO(jawilson): organize headers into a map
			// of headers off the key FILE_NAME_KEY so
			// this isn't O(N^2) where N is the number of
			// headers

			for _, filename := range files {
				header := find_header(headers, filename)
				if header == nil {
					panic("File not found in archive: " + filename)
				}
				copy_bytes(
					IOInfo{
						filename: filename,
					},
					IOInfo{
						file:        archive,
						seek_offset: as_int64(header[START_KEY]),
						size:        as_int64(header[SIZE_KEY]),
					})
			}
		})
}

func extract_command(args []string) {
	extract_files_by_predicate(args,
		func(header map[string]string) bool {
			return has_key(header, FILE_NAME_KEY)
		})
}

func extract_files_by_predicate(args []string, predicate func(map[string]string) bool) {
	for _, archive_name := range args {
		with_archive(archive_name,
			func(archive *os.File) {
				headers := read_headers(archive)
				for _, header := range headers {
					if predicate(header) {
						copy_bytes(
							IOInfo{
								filename: header[FILE_NAME_KEY],
							},
							IOInfo{
								file:        archive,
								seek_offset: as_int64(header[START_KEY]),
								size:        as_int64(header[SIZE_KEY]),
							})
					}
				}
			})
	}
}

// Call a handler function with the open file representing the named
// archive. The file is automatically closed when the handler returns
func with_archive(archive_name string, handler func(*os.File)) {
	archive, err := os.Open(archive_name)
	if err != nil {
		panic(err)
	}
	handler(archive)
	if err := archive.Close(); err != nil {
		panic(err)
	}
}

// Find the header for a paritcular file. Does not yet handle
// versioned files.
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
// "start:" such that their raw contents don't overlap (or of course
// overlap with a header) while minimizing wasted space.
//
// We don't really know where the first file should start without
// computing the size of all of the headers and that presents a
// problem since the start offset itself is technically a variable
// width quantity. To work around this, we first compute the size of
// all of the headers using a fixed width size string (00000000) and
// then as long as the actual headers when written also left pads the
// hexidecimal size to the same number of digits then we know how big
// each header really is. We also need to add one since we always add
// a blank "header" (a zero byte) according to the specification (this
// makes is much easier to determine where the last header is).
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
func write_archive(archive_name string, headers []map[string]string, inputs []IOInfo) {
	/* First we need to figure out where everything goes */
	layout_archive(headers)

	/* Open the output file */
	output, err := os.Create(archive_name)
	if err != nil {
		panic(err)
	}

	/* First write all of the headers */
	for _, member := range headers {
		if _, err := output.Write(header_to_bytes(member)); err != nil {
			panic(err)
		}
	}

	/* Write and empty header / zero byte to signal the end of headers. */
	if _, err := output.Write([]byte{0}); err != nil {
		panic(err)
	}

	/* Now write all of the raw data contents */
	for j, member := range headers {
		if as_int64(member[SIZE_KEY]) > 0 {
			copy_bytes(
				IOInfo{
					file: output,
				},
				inputs[j])
		}
	}

	/* Close the output file */
	if err := output.Close(); err != nil {
		panic(err)
	}
}

//
// This is a debugging routine that creates a textual version of a
// header to show a user.
//
func header_to_string(header map[string]string) string {
	result := ""
	visit_by_sorted_key(header,
		func(key string, value string) {
			result += key
			result += value
			result += "\n"
		})
	return result
}

//
// Convert the represetation of a header to the file-system disk byte
// format.
//
// A header always ends with a byte of zero which is an empty string
// and this routine always emits such an empty line.
//
func header_to_bytes(header map[string]string) []byte {
	result := []byte{}
	visit_by_sorted_key(header,
		func(key string, value string) {
			result = append(result, key_value_pair_to_bytes(key, value)...)
		})
	result = append(result, 0)
	return result
}

func key_value_pair_to_bytes(key string, value string) []byte {
	result := []byte{}
	result = append(result, []byte(key)...)
	result = append(result, []byte(value)...)
	result = append(result, 0)
	return result
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
		if verbosity >= VERBOSITY_INFO {
			fmt.Println(header_to_string(header))
		}
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
		str := read_string(archive)
		if len(str) == 0 {
			break
		}
		key_end := strings.Index(str, ":") + 1
		result[str[0:key_end]] = str[key_end:]
	}
	return result
}

// Read a UTF-8 string until a null byte is encountered.
func read_string(archive *os.File) string {
	bytes := make([]byte, 0)
	for {
		b := read_byte(archive)
		if b == 0 {
			return string(bytes)
		}
		bytes = append(bytes, b)
	}
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
func copy_bytes(out_info IOInfo, in_info IOInfo) {

	if verbosity >= VERBOSITY_INFO {
		fmt.Sprintf("Copy from %s to %s\n", in_info, out_info)
	}

	var input *os.File
	var output *os.File

	if in_info.file != nil {
		input = in_info.file
	} else {
		in, err := os.Open(in_info.filename)
		if err != nil {
			panic(err)
		}
		input = in
	}

	if in_info.seek_offset > 0 {
		offset, err := input.Seek(in_info.seek_offset, 0)
		if err != nil {
			panic(err)
		}
		if offset != in_info.seek_offset {
			panic("failed to seek to correct position")
		}
	}

	if out_info.file != nil {
		output = out_info.file
	} else {
		// open output file
		create_parent_directories(out_info.filename)
		output_foo, err := os.Create(out_info.filename)
		if err != nil {
			panic(err)
		}
		output = output_foo
	}

	copy_bytes_to_output(output, input, in_info.size)

	if in_info.file == nil {
		if err := input.Close(); err != nil {
			panic(err)
		}
	}

	if out_info.file == nil {
		if err := output.Close(); err != nil {
			panic(err)
		}
	}
}

//
// Copy num_bytes from an input file to an output file as efficiently
// as possbile.
//
func copy_bytes_to_output(output *os.File, input *os.File, num_bytes int64) {
	buffer := make([]byte, BUFFER_SIZE)
	for num_bytes > 0 {
		if num_bytes < int64(BUFFER_SIZE) {
			buffer = buffer[:num_bytes]
		}
		n, err := input.Read(buffer)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			panic("should always read at least one byte")
		}
		if _, err := output.Write(buffer); err != nil {
			panic(err)
		}
		num_bytes -= int64(n)
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

//
// Examine a single header and return non-localized errors and
// warnings.
//
func validate_header(header map[string]string) []string {
	result := []string{}

	if !is_present(header, SIZE_KEY) {
		result = append(result, "ERROR: A header does not have the required key -- size:")
	}

	if is_present(header, ALIGN_KEY) {
		result = append(result, "WARNING: This tool can doesn't respect alignment")
	}

	if is_present(header, FILE_VERSION_KEY) {
		result = append(result, "WARNING: This tool can doesn't handle multiple versions")
	}

	if is_present(header, DATA_COMPRESSION_ALGORITHM_KEY) !=
		// bug, should be DATA_SIZE_KEY
		is_present(header, DATA_COMPRESSION_ALGORITHM_KEY) {
		result = append(result, "ERROR: FOO and BAR must match")
	}

	// TODO:(jawilson): validate the layout which obviously can't be done here.

	return result
}

func is_present(m map[string]string, key string) bool {
	if _, ok := m[key]; ok {
		return true
	} else {
		return false
	}
}

//
// Visit the keys value pairs of this map according to the the
// "natural" sort order of the keys.
//
func visit_by_sorted_key(m map[string]string, visitor func(key string, value string)) {
	keys := sorted_keys(m)
	for _, key := range keys {
		visitor(key, m[key])
	}
}

//
// Returns the keys of a map according to a sort function.
//
func sorted_keys(m map[string]string) []string {
	result := []string{}
	for key, _ := range m {
		result = append(result, key)
	}

	sort.Strings(result)

	return result
}

// Return true if the given key is present in a header (even if it's
// value is the empty string)
func has_key(ht map[string]string, key string) bool {
	_, is_present := ht[key]
	return is_present
}

// Output the usage for this tool.
func usage() {
	fmt.Println(`Usage:    
core-archive create {core-archive-filename} [filenames...]
core-archive extract {core-archive-filename}
core-archive extract-by-file-name {core-archive-filename} [filenames...]
core-archive append [output archive] [archive 0] ...
core-archive list [archive 0] [archive 1] ...
core-archive headers [archive 0] [archive 1] ...
core-archive remove-by-file-name [archive 0] [filenames...]
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
	case "append":
		append_command(command_args)
	case "create":
		create_command(command_args)
	case "extract":
		extract_command(command_args)
	case "extract-by-file-name":
		extract_by_file_name_command(command_args)
	case "list":
		list_command(command_args)
	case "headers":
		headers_command(command_args)
	case "remove-by-file-name":
		remove_by_filename_command(command_args)
	default:
		usage()
	}
}
