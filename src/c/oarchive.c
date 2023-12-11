/**
 * @file oarchive.c
 *
 * This program implements a tool for creating, listing, appending,
 * and extracting archives in the Omni Archive File Format.
 */

#include <stdlib.h>

#define ARMYKNIFE_LIB_DEFAULT_LOG_LEVEL LOGGER_DEBUG
#define C_ARMYKNIFE_LIB_IMPL
#include "c-armyknife-lib-no-lines.h"

#if 0

// Non thread-safe macro/builder pattern.

enum over_write_t {
  OVERWRITE_NO,
  OVERWRITE_YES,
  OVERWRITE_ASK
};

enum is_pipeline_t {
  PIPELINE_NO,
  PIPELINE_YES,
  PIPELINE_AUTO,
}

boolean_t FLAG_verbose = 1;
over_write_t FLAG_overwrite = OVERWRITE_NO;
is_pipeline_t FLAG_is_pipeline = PIPELINE_AUTO;
char* FLAG_output_file = NULL;

void configure_command_line_parser() {
  flag_program_name("oarchive");

  flag_description("oarchive can create, list, extract, append to, or join archives in");
  flag_description("the omni archive format (`.oar`).");

  // Global Flags
  flag_boolean("--verbose", &FLAG_verbose);
  flag_description("When true, output more information useful for debugging");

  flag_enum("--overwrite" &FLAG_overwrite);
  flag_description("...");
  flag_enum_boolean_values(OVERWRITE_NO, OVERWRITE_YES);
  flag_enum_value("ask", OVERWRITE_ASK);

  flag_enum("--is-pipeline", &FLAG_is_pipline);
  flag_description("...");
  flag_enum_boolean_values(PIPELINE_NO, PIPELINE_YES);
  flag_enum_value("auto", PIPELINE_AUTO);

  configure_create_command();
  configure_list_command();
  configure_extract_command();
}

void configure_create_command() {
  flag_command("create");
  flag_abbreviation("c");

  flag_description("create an archive from the given files");
  
  flag_file_args(&FLAG_files);

  flag_string("--output-file", &FLAG_output_file);
  flag_abbreviation("--output");
  flag_abbreviation("-o");
}

buffer_t* error = parse_command_file(false, argc, argv);
if (error) {
  print_help(error);
  exit(1);
}

// TODO(jawilson): custom parsers...
// flag_uint64();
// flag_int64();
// flag_custom();

// Might not need right away...
// flag_add_to_command()
// flag_clear_command()

#endif

value_array_t* get_command_line_command_descriptors() {
  value_array_t* result = make_value_array(1);
  value_array_add(result, ptr_to_value(make_command_line_command_descriptor(
                              "create",
                              "create an archive from the given files (but not "
                              "directories currently")));
  value_array_add(result,
                  ptr_to_value(make_command_line_command_descriptor(
                      "list", "list all the members that have a filename")));
  value_array_add(
      result,
      ptr_to_value(make_command_line_command_descriptor(
          "extract", "extract all of the members that have a filename")));
  value_array_add(result, ptr_to_value(make_command_line_command_descriptor(
                              "append", "combine archives")));
  return result;
}

value_array_t* get_command_line_flag_descriptors() {
  value_array_t* result = make_value_array(1);
  value_array_add(
      result,
      ptr_to_value(make_command_line_flag_descriptor(
          "input-file", command_line_flag_type_string,
          "Specifies which archive to operate on for read operations")));
  value_array_add(result,
                  ptr_to_value(make_command_line_flag_descriptor(
                      "output-file", command_line_flag_type_string,
                      "Specifies the name of the archive output file name")));
  value_array_add(
      result,
      ptr_to_value(make_command_line_flag_descriptor(
          "output-directory", command_line_flag_type_string,
          "Specifies the directory where to place the output results")));
  return result;
}

command_line_parser_configuation_t* get_command_line_parser_config() {
  command_line_parser_configuation_t* config
      = malloc_struct(command_line_parser_configuation_t);
  config->program_name = "oarchive";
  config->program_description
      = "This is the pure C version of the Omni Archive Tool (most similar to "
        "ar or tar))";
  config->command_descriptors = get_command_line_command_descriptors();
  config->flag_descriptors = get_command_line_flag_descriptors();
  return config;
}

void append_header_and_file_contents(FILE* out, char* filename) {
  buffer_t* contents = make_buffer(1);
  contents = buffer_append_file_contents(contents, filename);
  fprintf(out, "filename=%s", filename);
  fputc(0, out);
  fprintf(out, "size=%d", contents->length);
  fputc(0, out);
  fputc(0, out);
  for (uint64_t i = 0; i < contents->length; i++) {
    fputc(buffer_get(contents, i), out);
  }
}

string_tree_t* read_header(FILE* in) {
  string_tree_t* metadata = NULL;
  while (!feof(in)) {
    if (file_peek_byte(in) == '\0') {
      fgetc(in);
      break;
    }
    // TODO(jawilson): If there is an illegal header line without an
    // =, this won't work very well.
    buffer_t* key = make_buffer(8);
    key = buffer_read_until(key, in, '=');
    buffer_t* value = make_buffer(8);
    value = buffer_read_until(value, in, '\0');
    if (key->length == 0 && value->length == 0) {
      return metadata;
    }
    metadata = string_tree_insert(metadata, buffer_to_c_string(key),
                                  str_to_value(buffer_to_c_string(value)));
  }
  return metadata;
}

/**
 * @typedef
 *
 * Defines the callback type signature for stream_members (which is
 * used for processing an archive while streaming it).
 */
typedef boolean_t (*stream_headers_callback_t)(FILE* input,
                                               string_tree_t* metadata,
                                               int64_t size,
                                               void* callback_data);

void log_metadata(string_tree_t* metadata) {
  log_info("Logging metadata");
  if (should_log_info()) {
    string_tree_foreach(metadata, key, value,
                        { log_info("'%s' = '%s'", key, value.str); });
  }
}

/**
 * A callback based interface for processing an archive while streaming
 * it.
 */
void stream_members(FILE* in, stream_headers_callback_t callback,
                    void* callback_data) {
  while (!file_eof(in)) {
    string_tree_t* metadata = read_header(in);
    log_metadata(metadata);
    int64_t size = 0;
    value_result_t size_value = string_tree_find(metadata, "size");
    if (!is_ok(size_value)) {
      log_warn("Encounterd a header without an explicit size.");
    } else {
      value_result_t data_size = string_parse_uint64_dec(size_value.str);
      if (!is_ok(data_size)) {
        log_fatal("Encounterd a header with an unparseable size %s",
                  size_value.str);
        fatal_error(ERROR_FATAL);
      } else {
        size = data_size.u64;
      }
    }

    // ---------------------------------------------------------------
    boolean_t skip_data = callback(in, metadata, size, callback_data);
    // ---------------------------------------------------------------

    if (skip_data && size > 0) {
      log_info("Skipping %lu\n", size);
      fseek(in, size, SEEK_CUR);
    }
  }
}

void create_command(command_line_parse_result_t args_and_files) {
  log_info("create_command");

  FILE* out = stdout;
  value_result_t output_filename_value
      = string_ht_find(args_and_files.flags, "output-file");
  if (is_ok(output_filename_value)) {
    out = fopen(output_filename_value.str, "w");
  }

  for (int i = 0; i < args_and_files.files->length; i++) {
    append_header_and_file_contents(
        stdout, value_array_get(args_and_files.files, i).str);
  }

  if (is_ok(output_filename_value)) {
    fclose(out);
  }
}

boolean_t list_command_callback(FILE* input, string_tree_t* metadata,
                                int64_t size, void* callback_data) {
  value_result_t filename_value = string_tree_find(metadata, "filename");
  if (is_ok(filename_value)) {
    fprintf(stdout, "%s\n", filename_value.str);
  } else {
    fprintf(stdout, "%s\n", "<<member lacks filename>>");
  }
  return true;
}

void list_command(command_line_parse_result_t args_and_files) {
  log_info("list_command");
  FILE* in = stdin;

  value_result_t input_filename_value
      = string_ht_find(args_and_files.flags, "input-file");

  if (is_ok(input_filename_value)) {
    log_info("opening %s", input_filename_value.str);
    // TODO(jawilson): safe file_open instead.
    in = fopen(input_filename_value.str, "r");
  }

  stream_members(in, &list_command_callback, NULL);

  if (is_ok(input_filename_value)) {
    fclose(in);
  }
}

boolean_t extract_command_callback(FILE* input, string_tree_t* metadata,
                                   int64_t size, void* callback_data) {
  command_line_parse_result_t args_and_files
      = *((command_line_parse_result_t*) callback_data);
  value_result_t filename_value = string_tree_find(metadata, "filename");
  if (is_ok(filename_value)) {
    log_info("Extracting %s", filename_value.str);
    FILE* output = fopen(filename_value.str, "w");
    file_copy_stream(input, output, false, size);
    fclose(output);
  }
  return false;
}

void extract_command(command_line_parse_result_t args_and_files) {
  log_info("list_command");
  FILE* in = stdin;

  value_result_t input_filename_value
      = string_ht_find(args_and_files.flags, "input-file");

  if (is_ok(input_filename_value)) {
    log_info("opening %s", input_filename_value.str);
    // TODO(jawilson): safe file_open instead.
    in = fopen(input_filename_value.str, "r");
  }

  stream_members(in, &extract_command_callback, &args_and_files);

  if (is_ok(input_filename_value)) {
    fclose(in);
  }
}

int main(int argc, char** argv) {
  configure_fatal_errors((fatal_error_config_t){
      .catch_sigsegv = true,
  });
  logger_init();
  command_line_parse_result_t args_and_files
      = parse_command_line(argc, argv, get_command_line_parser_config());

  if (args_and_files.command == NULL) {
    fatal_error(ERROR_BAD_COMMAND_LINE);
  }

  if (string_equal("create", args_and_files.command)) {
    create_command(args_and_files);
  } else if (string_equal("list", args_and_files.command)) {
    list_command(args_and_files);
  } else if (string_equal("extract", args_and_files.command)) {
    extract_command(args_and_files);
  } else {
    fatal_error(ERROR_BAD_COMMAND_LINE);
  }

  exit(0);
}
