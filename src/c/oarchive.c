/**
 * @file oarchive.c
 *
 * This program implements a tool for creating, listing, appending,
 * and extracting archives in the Omni Archive File Format.
 */

#include <stdlib.h>

#define C_ARMYKNIFE_LIB_IMPL
#include "c-armyknife-lib.h"

value_array_t* get_command_line_command_descriptors() {
  value_array_t* result = make_value_array(1);
  value_array_add(result, ptr_to_value(make_command_line_command_descriptor(
                              "create", "create an archive from the given files (but not directories currently")));
  value_array_add(result,
                  ptr_to_value(make_command_line_command_descriptor(
                      "list", "list all the members that have a filename")));
  value_array_add(result,
                  ptr_to_value(make_command_line_command_descriptor(
                      "extract", "extract all of the members that have a filename")));
  value_array_add(result,
                  ptr_to_value(make_command_line_command_descriptor(
                      "append", "combine archives")));
  return result;
}

value_array_t* get_command_line_flag_descriptors() {
  value_array_t* result = make_value_array(1);
  value_array_add(result,
                  ptr_to_value(make_command_line_flag_descriptor(
                      "input-file", command_line_flag_type_string,
                      "Specifies which archive to operate on for read operations")));
  value_array_add(result,
                  ptr_to_value(make_command_line_flag_descriptor(
                      "output-file", command_line_flag_type_string,
                      "Specifies the name of the archive output file name")));
  value_array_add(result,
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
      = "This is the pure C version of the Omni Archive Tool (most similar to ar or tar))";
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

void create_command(command_line_parse_result_t args_and_files) {
  FILE* out = stdout;
  value_result_t output_filename_value = string_ht_find(args_and_files.flags, "output-file");
  if (is_ok(output_filename_value)) {
      out = fopen(output_filename_value.str, "w");
  }

  for (int i = 0; i < args_and_files.files->length; i++) {
    append_header_and_file_contents(stdout, value_array_get(args_and_files.files, i).str);
  }

  if (is_ok(output_filename_value)) {
    fclose(out);
  }
}

int main(int argc, char** argv) {
  configure_fatal_errors((fatal_error_config_t){
      .catch_sigsegv = true,
    });
  command_line_parse_result_t args_and_files
      = parse_command_line(argc, argv, get_command_line_parser_config());

  if (args_and_files.command == NULL) {
    fatal_error(ERROR_BAD_COMMAND_LINE);
  }

  if (string_equal("create", args_and_files.command)) {
    create_command(args_and_files);
  }

  exit(0);
}

