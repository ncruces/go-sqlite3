#include <stddef.h>

#include "sql3parse_table.c"
#include "libc.c"
#include "malloc_bump.c"

static_assert(sizeof(sql3table) == 64, "Unexpected size");
static_assert(offsetof(sql3table, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3table, schema) == 8, "Unexpected offset");
static_assert(offsetof(sql3table, comment) == 16, "Unexpected offset");
static_assert(offsetof(sql3table, is_temporary) == 24, "Unexpected offset");
static_assert(offsetof(sql3table, is_ifnotexists) == 25, "Unexpected offset");
static_assert(offsetof(sql3table, is_withoutrowid) == 26, "Unexpected offset");
static_assert(offsetof(sql3table, is_strict) == 27, "Unexpected offset");
static_assert(offsetof(sql3table, num_columns) == 28, "Unexpected offset");
static_assert(offsetof(sql3table, columns) == 32, "Unexpected offset");
static_assert(offsetof(sql3table, num_constraint) == 36, "Unexpected offset");
static_assert(offsetof(sql3table, constraints) == 40, "Unexpected offset");
static_assert(offsetof(sql3table, type) == 44, "Unexpected offset");
static_assert(offsetof(sql3table, current_name) == 48, "Unexpected offset");
static_assert(offsetof(sql3table, new_name) == 56, "Unexpected offset");

static_assert(sizeof(sql3tableconstraint) == 28, "Unexpected size");
static_assert(offsetof(sql3tableconstraint, type) == 0, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, name) == 4, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, num_indexed) == 12, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, indexed_columns) == 16, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, conflict_clause) == 20, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, is_autoincrement) == 24, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, check_expr) == 12, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, foreignkey_num) == 12, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, foreignkey_name) == 16, "Unexpected offset");
static_assert(offsetof(sql3tableconstraint, foreignkey_clause) == 20, "Unexpected offset");

static_assert(sizeof(sql3column) == 148, "Unexpected size");
static_assert(offsetof(sql3column, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3column, type) == 8, "Unexpected offset");
static_assert(offsetof(sql3column, length) == 16, "Unexpected offset");
static_assert(offsetof(sql3column, comment) == 24, "Unexpected offset");
static_assert(offsetof(sql3column, is_primarykey) == 32, "Unexpected offset");
static_assert(offsetof(sql3column, is_autoincrement) == 33, "Unexpected offset");
static_assert(offsetof(sql3column, is_notnull) == 34, "Unexpected offset");
static_assert(offsetof(sql3column, is_unique) == 35, "Unexpected offset");
static_assert(offsetof(sql3column, pk_constraint_name) == 36, "Unexpected offset");
static_assert(offsetof(sql3column, pk_order) == 44, "Unexpected offset");
static_assert(offsetof(sql3column, pk_conflictclause) == 48, "Unexpected offset");
static_assert(offsetof(sql3column, notnull_constraint_name) == 52, "Unexpected offset");
static_assert(offsetof(sql3column, notnull_conflictclause) == 60, "Unexpected offset");
static_assert(offsetof(sql3column, unique_constraint_name) == 64, "Unexpected offset");
static_assert(offsetof(sql3column, unique_conflictclause) == 72, "Unexpected offset");
static_assert(offsetof(sql3column, num_check_constraints) == 76, "Unexpected offset");
static_assert(offsetof(sql3column, check_constraints) == 80, "Unexpected offset");
static_assert(offsetof(sql3column, default_constraint_name) == 84, "Unexpected offset");
static_assert(offsetof(sql3column, default_expr) == 92, "Unexpected offset");
static_assert(offsetof(sql3column, collate_constraint_name) == 100, "Unexpected offset");
static_assert(offsetof(sql3column, collate_name) == 108, "Unexpected offset");
static_assert(offsetof(sql3column, foreignkey_constraint_name) == 116, "Unexpected offset");
static_assert(offsetof(sql3column, foreignkey_clause) == 124, "Unexpected offset");
static_assert(offsetof(sql3column, generated_constraint_name) == 128, "Unexpected offset");
static_assert(offsetof(sql3column, generated_expr) == 136, "Unexpected offset");
static_assert(offsetof(sql3column, generated_type) == 144, "Unexpected offset");

static_assert(sizeof(sql3checkconstraint) == 16, "Unexpected size");
static_assert(offsetof(sql3checkconstraint, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3checkconstraint, expr) == 8, "Unexpected offset");

static_assert(sizeof(sql3foreignkey) == 36, "Unexpected size");
static_assert(offsetof(sql3foreignkey, table) == 0, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, num_columns) == 8, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, column_name) == 12, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, on_delete) == 16, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, on_update) == 20, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, match) == 24, "Unexpected offset");
static_assert(offsetof(sql3foreignkey, deferrable) == 32, "Unexpected offset");

static_assert(sizeof(sql3idxcolumn) == 20, "Unexpected size");
static_assert(offsetof(sql3idxcolumn, name) == 0, "Unexpected offset");
static_assert(offsetof(sql3idxcolumn, collate_name) == 8, "Unexpected offset");
static_assert(offsetof(sql3idxcolumn, order) == 16, "Unexpected offset");

static_assert(sizeof(sql3string) == 8, "Unexpected size");
static_assert(offsetof(sql3string, ptr) == 0, "Unexpected offset");
static_assert(offsetof(sql3string, length) == 4, "Unexpected offset");
