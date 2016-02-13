#include <sys/timerfd.h>
#include <stdint.h>
#include <unistd.h>
#include "interval.h"

int interval_create(uint64_t ns) {
	const struct timespec tspec = {
		.tv_sec = ns / 1000000000,
		.tv_nsec = ns % 1000000000,
	};
	const struct itimerspec ispec = {
		.it_interval = tspec,
		.it_value = tspec,
	};

	int fd = timerfd_create(CLOCK_MONOTONIC, TFD_CLOEXEC);
	if (fd == -1) {
		return -1;
	}
	if (timerfd_settime(fd, 0, &ispec, NULL)) {
		close(fd);
		return -1;
	}
	return fd;
}

uint64_t interval_wait(int fd) {
	uint64_t count;

	if (read(fd, &count, sizeof(count)) < sizeof(count)) {
		return 0;
	}
	return count;
}
