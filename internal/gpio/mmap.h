/*
 * biscornu -  An odd little thing, covered with embroidery
 *
 * Copyright (C) 2016 Benjamin Gilbert
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of version 3 of the GNU General Public License as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

#ifndef GPIO_MMAP_H
#define GPIO_MMAP_H

#include <sys/types.h>
#include <stdint.h>

struct range *range_map(off_t base, size_t len);
void range_unmap(struct range *range);
uint32_t range_get_u32(const struct range *range, size_t off);
void range_set_u32(const struct range *range, size_t off, uint32_t val);

#endif
