#include <stddef.h>

#include "sql3parse_table.c"

static_assert(offsetof(sql3table, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3table, schema) == 8, "Unexpected offset");
static_assert(offsetof(sql3table, comment) == 16, "Unexpected offset");
static_assert(offsetof(sql3table, is_temporary) == 24, "Unexpected offset");
static_assert(offsetof(sql3table, is_ifnotexists) == 25, "Unexpected offset");
static_assert(offsetof(sql3table, is_withoutrowid) == 26, "Unexpected offset");
static_assert(offsetof(sql3table, is_strict) == 27, "Unexpected offset");
static_assert(offsetof(sql3table, num_columns) == 28, "Unexpected offset");
static_assert(offsetof(sql3table, columns) == 32, "Unexpected offset");
static_assert(offsetof(sql3table, type) == 44, "Unexpected offset");
static_assert(offsetof(sql3table, current_name) == 48, "Unexpected offset");
static_assert(offsetof(sql3table, new_name) == 56, "Unexpected offset");

static_assert(offsetof(sql3column, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3column, type) == 8, "Unexpected offset");
static_assert(offsetof(sql3column, length) == 16, "Unexpected offset");
static_assert(offsetof(sql3column, constraint_name) == 24, "Unexpected offset");
static_assert(offsetof(sql3column, comment) == 32, "Unexpected offset");
static_assert(offsetof(sql3column, is_primarykey) == 40, "Unexpected offset");
static_assert(offsetof(sql3column, is_autoincrement) == 41, "Unexpected offset");
static_assert(offsetof(sql3column, is_notnull) == 42, "Unexpected offset");
static_assert(offsetof(sql3column, is_unique) == 43, "Unexpected offset");
static_assert(offsetof(sql3column, pk_order) == 44, "Unexpected offset");
static_assert(offsetof(sql3column, pk_conflictclause) == 48, "Unexpected offset");
static_assert(offsetof(sql3column, notnull_conflictclause) == 52, "Unexpected offset");
static_assert(offsetof(sql3column, unique_conflictclause) == 56, "Unexpected offset");
static_assert(offsetof(sql3column, check_expr) == 60, "Unexpected offset");
static_assert(offsetof(sql3column, default_expr) == 68, "Unexpected offset");
static_assert(offsetof(sql3column, collate_name) == 76, "Unexpected offset");
static_assert(offsetof(sql3column, foreignkey_clause) == 84, "Unexpected offset");

static_assert(offsetof(sql3foreignkey, table) == 0, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, num_columns) == 8, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, column_name) == 12, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, on_delete) == 16, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, on_update) == 20, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, match) == 24, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, deferrable) == 32, "Unexpected offset");