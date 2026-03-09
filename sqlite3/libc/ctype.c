#include <ctype.h>

int(isalnum)(int c) { return isalpha(c) || isdigit(c); }
int(isalpha)(int c) { return ((unsigned)c | 32) - 'a' < 26; }
int(isascii)(int c) { return !(c & ~0x7f); }
int(isblank)(int c) { return (c == ' ' || c == '\t'); }
int(iscntrl)(int c) { return (unsigned)c < 0x20 || c == 0x7f; }
int(isdigit)(int c) { return (unsigned)c - '0' < 10; }
int(isgraph)(int c) { return (unsigned)c - 0x21 < 0x5e; }
int(islower)(int c) { return (unsigned)c - 'a' < 26; }
int(isprint)(int c) { return (unsigned)c - 0x20 < 0x5f; }
int(ispunct)(int c) { return isgraph(c) && !isalnum(c); }
int(isspace)(int c) { return c == ' ' || (unsigned)c - '\t' < 5; }
int(isupper)(int c) { return (unsigned)c - 'A' < 26; }
int(isxdigit)(int c) { return isdigit(c) || ((unsigned)c | 32) - 'a' < 6; }

int(tolower)(int c) { return isupper(c) ? c | 32 : c; }
int(toupper)(int c) { return islower(c) ? c & 0x5f : c; }
