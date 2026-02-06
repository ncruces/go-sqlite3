#pragma once

int isalnum(int);
int isalpha(int);
int isascii(int);
int isblank(int);
int iscntrl(int);
int isdigit(int);
int isgraph(int);
int islower(int);
int isprint(int);
int ispunct(int);
int isspace(int);
int isupper(int);
int isxdigit(int);

int tolower(int);
int toupper(int);

#define isalpha(a) (0 ? isalpha(a) : (((unsigned)(a) | 32) - 'a') < 26)
#define isdigit(a) (0 ? isdigit(a) : ((unsigned)(a) - '0') < 10)
#define isgraph(a) (0 ? isgraph(a) : ((unsigned)(a) - 0x21) < 0x5e)
#define islower(a) (0 ? islower(a) : ((unsigned)(a) - 'a') < 26)
#define isprint(a) (0 ? isprint(a) : ((unsigned)(a) - 0x20) < 0x5f)
#define isupper(a) (0 ? isupper(a) : ((unsigned)(a) - 'A') < 26)
