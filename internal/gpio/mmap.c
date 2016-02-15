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

#include <sys/types.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <unistd.h>
#include "mmap.h"

struct range {
	volatile uint32_t *base;
	size_t len;
};

static bool off_ok(const struct range *range, size_t off) {
	return off + sizeof(uint32_t) <= range->len &&
		!(off & (sizeof(uint32_t) - 1));
}

struct range *range_map(off_t base, size_t len) {
	int fd = open("/dev/mem", O_RDWR);
	if (fd == -1) {
		return NULL;
	}

	void *mapping = mmap(NULL, len, PROT_READ|PROT_WRITE, MAP_SHARED, fd, base);
	close(fd);
	if (mapping == NULL) {
		return NULL;
	}

	struct range *range = calloc(1, sizeof(struct range));
	range->base = mapping;
	range->len = len;
	return range;
}

void range_unmap(struct range *range) {
	munmap((void *) range->base, range->len);
	free(range);
}

uint32_t range_get_u32(const struct range *range, size_t off) {
	if (!off_ok(range, off)) {
		return 0;
	}
	return range->base[off / sizeof(uint32_t)];
}

void range_set_u32(const struct range *range, size_t off, uint32_t val) {
	if (!off_ok(range, off)) {
		return;
	}
	range->base[off / sizeof(uint32_t)] = val;
}
