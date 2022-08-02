package main

import (
        "fmt"
        "io"
        "os"
        "strconv"
        "strings"
)

const (
        NAME_PROPERTY  = "file-name:"
        SIZE_PROPERTY  = "size:"
        START_PROPERTY = "start:"
)

func create_command(args []string) {
        archive_name := args[0]
        files := args[1:]

        headers := []map[string]string{}

        for _, member := range files {
                fmt.Println("Adding " + member)
                header := make(map[string]string)
                header[NAME_PROPERTY] = member

                fi, err := os.Stat(member)
                if err != nil {
                        panic(err)
                }

                if fi.IsDir() {
                        panic("Can't handle directories yet")
                }

                header[SIZE_PROPERTY] = fmt.Sprintf("%x", fi.Size())

                headers = append(headers, header)
        }

        layout_archive(headers)
        write_archive(archive_name, headers)
}

func extract_files_command(args []string) {
        archive_name := args[0]
        files := args[1:]
        _ = files

        archive, err := os.Open(archive_name)
        if err != nil {
                panic(err)
        }

        members := read_headers(archive)
        _ = members

        if err := archive.Close(); err != nil {
                panic(err)
        }
}

//
// Assign START_PROPERTY to all members
//
func layout_archive(headers []map[string]string) {
        header_size := 0
        for _, member := range headers {
                member[START_PROPERTY] = "00000000"
                header_size += len(header_to_bytes(member))
        }
        header_size += 1
        start := int64(header_size)
        for _, member := range headers {
                if start > (1 << 31) {
                        panic("archive is currently to too large")
                }
                member[START_PROPERTY] = fmt.Sprintf("%08x", start)
                start += as_int64(member[SIZE_PROPERTY])
        }
}

func as_int64(value string) int64 {
        num, err := strconv.ParseInt(value, 16, 64)
        if err != nil {
                panic(err)
        }
        return num
}

func write_archive(archive_name string, headers []map[string]string) {
        /* Open the output file */
        fo, err := os.Create("output.txt")
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
                write_file_contents(fo, member[NAME_PROPERTY])
        }

        /* Close the output file */
        if err := fo.Close(); err != nil {
                panic(err)
        }
}

func write_file_contents(fo *os.File, filename string) {
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
                if _, err := fo.Write(buf[:n]); err != nil {
                        panic(err)
                }
        }
        if err := fi.Close(); err != nil {
                panic(err)
        }
}

func header_to_string(header map[string]string) string {
        result := ""
        for key, value := range header {
                result += key
                result += value
                result += "\n"
        }
        return result
}

func header_to_bytes(header map[string]string) []byte {
        result := []byte{}
        for key, value := range header {
                result = append(result, attribute_to_bytes(key, value)...)
        }
        result = append(result, 0)
        return result
}

func attribute_to_bytes(key string, value string) []byte {
        result := []byte{}
        result = append(result, []byte(key)...)
        result = append(result, []byte(value)...)
        result = append(uleb128(int64(len(result))), result...)
        return result
}

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

func read_uleb128(archive *os.File) int64 {
        result := int64(0)
        shift := 0
        for {
                b, err := read_byte(archive)
                if err != nil {
                        panic(err)
                }
                result |= int64((b & 0x7f)) << shift
                if b&(1<<7) == 0 {
                        break
                }
                shift += 7
        }
        // fmt.Printf("uleb128 is %d", result)
        return result
}

func read_byte(archive *os.File) (byte, error) {
        barray := make([]byte, 1)
        n, err := archive.Read(barray)
        if n != 1 {
                panic("Expected to read one byte")
        }
        return barray[0], err
}

func usage() {
        fmt.Println(`Usage:    
core-archive create [filenames...]
core-archive extract-all [directory]
core-archive extract-files [directory] [filenames...]
core-archive append [archive 0] [archive 1] ...
core-archive list [archive 0] [archive 1] ...
core-archive headers [archive 0] [archive 1] ...
core-archive remove [archive 0] [filenames...]
core-archive update [filenames...]
core-archive --usage
core-archive --version`)
}

func main() {
        first := os.Args[1]
        switch first {
        case "create":
                create_command(os.Args[2:])
        case "extract-files":
                extract_files_command(os.Args[2:])
        default:
                usage()
        }
}
