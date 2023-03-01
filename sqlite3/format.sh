#!/usr/bin/env bash
cd -P -- "$(dirname -- "$0")"

clang-format -i \
	main.c \
	os.c \
	qsort.c \
	amalg.c