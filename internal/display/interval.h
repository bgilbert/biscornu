#ifndef DISPLAY_TIMER_H
#define DISPLAY_TIMER_H

#include <stdint.h>

int interval_create(uint64_t ns);
uint64_t interval_wait(int fd);
void interval_destroy(int fd);

#endif
