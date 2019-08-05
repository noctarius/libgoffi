project(libgoffi-tests)
cmake_minimum_required(VERSION 3.6)

add_library(libgoffi_tests SHARED libtest.c)
target_link_libraries(libgoffi_tests m)
set_target_properties(libgoffi_tests PROPERTIES OUTPUT_NAME "goffitests")