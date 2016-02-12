#ifndef GPIO_MMAP_H
#define GPIO_MMAP_H

#include <sys/types.h>
#include <stdint.h>

struct range *range_map(off_t base, size_t len);
void range_unmap(struct range *range);
uint32_t range_get_u32(const struct range *range, size_t off);
void range_set_u32(const struct range *range, size_t off, uint32_t val);

#endif
