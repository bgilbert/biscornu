#include <stdint.h>
#include <unistd.h>
#include "interval.h"

uint64_t interval_wait(int fd) {
	uint64_t count;

	if (read(fd, &count, sizeof(count)) < sizeof(count)) {
		return 0;
	}
	return count;
}
