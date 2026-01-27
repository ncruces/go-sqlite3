#pragma once

double acos(double);
double acosh(double);
double asin(double);
double asinh(double);
double atan(double);
double atan2(double, double);
double atanh(double);
double ceil(double);
double cos(double);
double cosh(double);
double exp(double);
double fabs(double);
double floor(double);
double fmod(double, double);
double log(double);
double log10(double);
double log2(double);
double pow(double, double);
double sin(double);
double sinh(double);
double tan(double);
double tanh(double);
double trunc(double);
double sqrt(double);

#define sqrt(x) (__builtin_sqrt(x))
#define isnan(x) (__builtin_isnan(x))
