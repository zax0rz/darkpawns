package common

import "time"

// Asctime formats t the way C asctime() does, without its trailing newline:
// "%.3s %.3s%3d %.2d:%.2d:%.2d %d". The day of the month is padded to width
// 3 with the separating space, so the 5th prints "Nov  5" (two spaces).
// do_date and do_uptime (src/act.wizard.c:1814-1826) and the mail reader
// (src/mail.c:429) print it.
func Asctime(t time.Time) string {
	return t.Format("Mon Jan _2 15:04:05 2006")
}
