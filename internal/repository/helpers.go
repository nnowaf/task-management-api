package repository

import "time"

// scannable is satisfied by both pgx.Row and pgx.Rows, so a single scan helper
// serves single-row lookups and row iteration.
type scannable interface {
	Scan(dest ...any) error
}

func nowUTC() time.Time { return time.Now().UTC() }

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
