package timeutil

import (
	"fmt"
	"strings"
	"time"
)

const (
	LayoutLocal     = "2006-01-02 15:04"
	LayoutLocalSec  = "2006-01-02 15:04:05"
	DefaultTimezone = "Asia/Shanghai"
)

var Beijing *time.Location

func init() {
	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	Beijing = loc
}

func Now() time.Time {
	return time.Now().In(Beijing)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func Location(iana string) *time.Location {
	if strings.TrimSpace(iana) == "" {
		return Beijing
	}
	loc, err := time.LoadLocation(iana)
	if err != nil {
		return Beijing
	}
	return loc
}

func ParseCatchLocal(localStr, iana string) (time.Time, *time.Location, error) {
	loc := Location(iana)
	s := strings.TrimSpace(localStr)
	if s == "" {
		return time.Time{}, loc, fmt.Errorf("empty local time")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), t.Location(), nil
	}
	for _, layout := range []string{LayoutLocalSec, LayoutLocal} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t.UTC(), loc, nil
		}
	}
	return time.Time{}, loc, fmt.Errorf("invalid local time %q", localStr)
}

func FormatLocal(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = Beijing
	}
	return t.In(loc).Format(LayoutLocalSec)
}

func FormatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func CivilDate(t time.Time, loc *time.Location) (year int, month time.Month, day int) {
	if loc == nil {
		loc = Beijing
	}
	x := t.In(loc)
	return x.Year(), x.Month(), x.Day()
}
