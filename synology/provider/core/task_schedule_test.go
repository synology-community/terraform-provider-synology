package core

import "testing"

// The mappings asserted here were established against a live DSM 7.4 by
// creating disabled probe tasks and reading back what DSM stored and when it
// said the task would next run:
//
//	repeat_date 1001                 daily at hour:minute
//	repeat_date 1002 + week_day      weekly on those days at hour:minute
//	repeat_hour N (hour = start)     every N hours from hour:minute
//	repeat_min  N (minute = start)   every N minutes from :minute
//
// DSM validates none of these values -- hour=128 is accepted and stored, and
// the task simply never fires -- so every case below is the provider's
// responsibility, not the device's.

func TestParseScheduleDailyUsesLiteralHourAndMinute(t *testing.T) {
	for _, tc := range []struct {
		spec   string
		minute int64
		hour   int64
	}{
		{"17 3 * * *", 17, 3},
		{"0 7 * * *", 0, 7},
		{"30 2 * * *", 30, 2},
		{"0 0 * * *", 0, 0},     // bit 0 in both fields
		{"59 23 * * *", 59, 23}, // upper bound of both fields
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err != nil {
				t.Fatalf("parseSchedule(%q) returned error: %v", tc.spec, err)
			}
			if got.Minute != tc.minute {
				t.Errorf("minute = %d, want %d (a bitfield leaked through if this is 1<<%d)",
					got.Minute, tc.minute, tc.minute)
			}
			if got.Hour != tc.hour {
				t.Errorf("hour = %d, want %d (a bitfield leaked through if this is 1<<%d)",
					got.Hour, tc.hour, tc.hour)
			}
			if got.RepeatDate != dsmRepeatDaily {
				t.Errorf("repeat_date = %d, want %d (daily)", got.RepeatDate, dsmRepeatDaily)
			}
			if got.WeekDay != "0,1,2,3,4,5,6" {
				t.Errorf("week_day = %q, want all seven days", got.WeekDay)
			}
			if got.RepeatMin != 0 || got.RepeatHour != 0 {
				t.Errorf("repeat_min/repeat_hour = %d/%d, want 0/0 for a fixed daily time",
					got.RepeatMin, got.RepeatHour)
			}
		})
	}
}

// The day-of-week field used to be parsed and then discarded, so "0 3 * * 1"
// produced a task that ran every morning instead of on Mondays.
func TestParseScheduleWeekdaysSelectDSMWeeklyMode(t *testing.T) {
	for _, tc := range []struct {
		spec    string
		weekDay string
	}{
		{"17 3 * * 1", "1"},
		{"17 3 * * 1-5", "1,2,3,4,5"},
		{"17 3 * * 1,3", "1,3"},
		{"17 3 * * 0", "0"},
		{"17 3 * * 6", "6"},
		{"17 3 * * mon", "1"},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err != nil {
				t.Fatalf("parseSchedule(%q) returned error: %v", tc.spec, err)
			}
			if got.RepeatDate != dsmRepeatWeekly {
				t.Errorf("repeat_date = %d, want %d (weekly)", got.RepeatDate, dsmRepeatWeekly)
			}
			if got.WeekDay != tc.weekDay {
				t.Errorf("week_day = %q, want %q", got.WeekDay, tc.weekDay)
			}
			if got.Hour != 3 || got.Minute != 17 {
				t.Errorf("hour/minute = %d/%d, want 3/17", got.Hour, got.Minute)
			}
		})
	}
}

// "*" and "*/N" describe an interval, which DSM stores as repeat_min or
// repeat_hour with hour/minute carrying the starting point.
func TestParseScheduleIntervalsMapToRepeatCounters(t *testing.T) {
	for _, tc := range []struct {
		spec                  string
		minute, hour          int64
		repeatMin, repeatHour int64
	}{
		{"*/15 * * * *", 0, 0, 15, 0}, // every 15 minutes
		{"5/15 * * * *", 5, 0, 15, 0}, // every 15 minutes from :05
		{"* * * * *", 0, 0, 1, 0},     // every minute
		{"30 */6 * * *", 30, 0, 0, 6}, // every 6 hours at :30
		{"30 * * * *", 30, 0, 0, 1},   // every hour at :30
		{"30 2/6 * * *", 30, 2, 0, 6}, // every 6 hours from 02:30
		// A list whose values are evenly spaced and run to the end of the field
		// IS an interval: ":00 and :30" is every 30 minutes, and DSM stores it
		// as such. Rejecting these would refuse schedules the device supports.
		{"0,30 * * * *", 0, 0, 30, 0},
		{"0 3,15 * * *", 0, 3, 0, 12}, // every 12 hours from 03:00
		// Equivalent to */15: minute 45 plus a 15-minute step runs past 59.
		{"0-45/15 * * * *", 0, 0, 15, 0},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err != nil {
				t.Fatalf("parseSchedule(%q) returned error: %v", tc.spec, err)
			}
			if got.Minute != tc.minute || got.Hour != tc.hour {
				t.Errorf("hour/minute = %d/%d, want %d/%d",
					got.Hour, got.Minute, tc.hour, tc.minute)
			}
			if got.RepeatMin != tc.repeatMin || got.RepeatHour != tc.repeatHour {
				t.Errorf("repeat_min/repeat_hour = %d/%d, want %d/%d",
					got.RepeatMin, got.RepeatHour, tc.repeatMin, tc.repeatHour)
			}
		})
	}
}

// Weekly and interval combine: DSM accepted repeat_date 1002 with repeat_hour
// set, and scheduled the next run on the next selected weekday.
func TestParseScheduleWeeklyIntervalCombination(t *testing.T) {
	got, err := parseSchedule("30 */6 * * 1,3")
	if err != nil {
		t.Fatalf("parseSchedule returned error: %v", err)
	}
	if got.RepeatDate != dsmRepeatWeekly || got.WeekDay != "1,3" {
		t.Errorf("repeat_date/week_day = %d/%q, want %d/\"1,3\"",
			got.RepeatDate, got.WeekDay, dsmRepeatWeekly)
	}
	if got.RepeatHour != 6 || got.Minute != 30 {
		t.Errorf("repeat_hour/minute = %d/%d, want 6/30", got.RepeatHour, got.Minute)
	}
}

// Schedules DSM cannot represent must fail at plan/apply. Accepting them
// writes a task that looks healthy and never runs at the requested times.
func TestParseScheduleRejectsWhatDSMCannotExpress(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec string
	}{
		{"bounded hour range", "0 9-17 * * *"},
		{"bounded minute range", "10-20 3 * * *"},
		{"stepped range stopping early", "0-30/15 * * * *"},
		{"uneven minute list", "0,7,30 3 * * *"},
		{"minute interval inside one hour", "*/15 3 * * *"},
		{"day of month", "0 3 1 * *"},
		{"day-of-month range", "0 3 1-7 * *"},
		{"month restriction", "0 3 * 6 *"},
		{"minute interval with stepped hours", "*/15 */6 * * *"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err == nil {
				t.Fatalf("parseSchedule(%q) = %+v, want an error", tc.spec, got)
			}
		})
	}
}

// An empty schedule is how the resource signals "run once, now".
func TestParseScheduleEmptySpecIsNotAnError(t *testing.T) {
	got, err := parseSchedule("")
	if err != nil {
		t.Fatalf("parseSchedule(\"\") returned error: %v", err)
	}
	if got.RepeatDate != 0 {
		t.Errorf("repeat_date = %d, want the zero value", got.RepeatDate)
	}
}

// An unparseable expression must surface util's error rather than a zero
// schedule.
func TestParseScheduleInvalidSpecReturnsError(t *testing.T) {
	if _, err := parseSchedule("not a cron expression"); err == nil {
		t.Fatal("parseSchedule(\"not a cron expression\") returned no error")
	}
}

// The "@every ..." descriptors set only the repeat_* counters and leave every
// cron field empty; they must pass through untouched.
func TestParseScheduleEveryDescriptors(t *testing.T) {
	for _, tc := range []struct {
		spec                              string
		repeatMin, repeatHour, repeatDate int64
	}{
		{"@every 15m", 15, 0, dsmRepeatDaily},
		{"@every 6h", 0, 6, dsmRepeatDaily},
		{"@every 48h", 0, 0, 2},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err != nil {
				t.Fatalf("parseSchedule(%q) returned error: %v", tc.spec, err)
			}
			if got.RepeatMin != tc.repeatMin || got.RepeatHour != tc.repeatHour {
				t.Errorf("repeat_min/repeat_hour = %d/%d, want %d/%d",
					got.RepeatMin, got.RepeatHour, tc.repeatMin, tc.repeatHour)
			}
			if got.RepeatDate != tc.repeatDate {
				t.Errorf("repeat_date = %d, want %d", got.RepeatDate, tc.repeatDate)
			}
			if got.Hour != 0 || got.Minute != 0 {
				t.Errorf("hour/minute = %d/%d, want 0/0 for an interval schedule",
					got.Hour, got.Minute)
			}
		})
	}
}

// The named descriptors set hour and minute to bit 0, which must convert to
// midnight rather than to the literal 1.
func TestParseScheduleNamedDescriptors(t *testing.T) {
	for _, tc := range []struct {
		spec         string
		hour, minute int64
		repeatDate   int64
		weekDay      string
	}{
		{"@daily", 0, 0, dsmRepeatDaily, "0,1,2,3,4,5,6"},
		{"@midnight", 0, 0, dsmRepeatDaily, "0,1,2,3,4,5,6"},
		{"@weekly", 0, 0, dsmRepeatWeekly, "0"},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parseSchedule(tc.spec)
			if err != nil {
				t.Fatalf("parseSchedule(%q) returned error: %v", tc.spec, err)
			}
			if got.Hour != tc.hour || got.Minute != tc.minute {
				t.Errorf("hour/minute = %d/%d, want %d/%d",
					got.Hour, got.Minute, tc.hour, tc.minute)
			}
			if got.RepeatDate != tc.repeatDate {
				t.Errorf("repeat_date = %d, want %d", got.RepeatDate, tc.repeatDate)
			}
			if got.WeekDay != tc.weekDay {
				t.Errorf("week_day = %q, want %q", got.WeekDay, tc.weekDay)
			}
		})
	}
}

// @hourly and @yearly restrict fields DSM cannot express; they must fail
// loudly rather than be silently mis-stored.
func TestParseScheduleUnsupportedDescriptors(t *testing.T) {
	for _, spec := range []string{"@monthly", "@yearly", "@annually"} {
		t.Run(spec, func(t *testing.T) {
			if _, err := parseSchedule(spec); err == nil {
				t.Fatalf("parseSchedule(%q) returned no error", spec)
			}
		})
	}
}

func TestCronFieldValues(t *testing.T) {
	// Bit 0 and bit 3 set, plus the star bit, which must be masked off before
	// the values are read.
	// Converted through a variable: int64(cronStarBit) is a constant
	// conversion and overflows, which is why the parser does the same dance.
	star := cronStarBit
	mask := int64(1)<<0 | int64(1)<<3 | int64(star)
	got := cronFieldValues(mask, 23)
	if len(got) != 2 || got[0] != 0 || got[1] != 3 {
		t.Fatalf("cronFieldValues = %v, want [0 3]", got)
	}
	if v := cronFieldValues(0, 59); len(v) != 0 {
		t.Errorf("cronFieldValues(0) = %v, want empty", v)
	}
	// Values above max must not be reported.
	if v := cronFieldValues(int64(1)<<40, 23); len(v) != 0 {
		t.Errorf("cronFieldValues past max = %v, want empty", v)
	}
}

func TestClassifyCronField(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		f, err := classifyCronField(0, 59, "minute")
		if err != nil || f.start != 0 || f.step != 0 {
			t.Fatalf("got %+v, %v; want zero field and no error", f, err)
		}
	})
	t.Run("single value", func(t *testing.T) {
		f, err := classifyCronField(int64(1)<<17, 59, "minute")
		if err != nil || f.start != 17 || f.step != 0 {
			t.Fatalf("got %+v, %v; want start 17 step 0", f, err)
		}
	})
	t.Run("even interval to the end", func(t *testing.T) {
		var mask int64
		for _, v := range []uint{0, 15, 30, 45} {
			mask |= int64(1) << v
		}
		f, err := classifyCronField(mask, 59, "minute")
		if err != nil || f.start != 0 || f.step != 15 {
			t.Fatalf("got %+v, %v; want start 0 step 15", f, err)
		}
	})
	t.Run("uneven set is rejected", func(t *testing.T) {
		mask := int64(1)<<0 | int64(1)<<15 | int64(1)<<31
		if _, err := classifyCronField(mask, 59, "minute"); err == nil {
			t.Fatal("uneven values were accepted")
		}
	})
	t.Run("interval stopping early is rejected", func(t *testing.T) {
		mask := int64(1)<<0 | int64(1)<<10 | int64(1)<<20
		if _, err := classifyCronField(mask, 59, "minute"); err == nil {
			t.Fatal("bounded window was accepted")
		}
	})
}

func TestCronFieldString(t *testing.T) {
	if got := cronFieldString([]int64{1, 3, 5}); got != "1,3,5" {
		t.Errorf("cronFieldString = %q, want \"1,3,5\"", got)
	}
	if got := cronFieldString(nil); got != "" {
		t.Errorf("cronFieldString(nil) = %q, want empty", got)
	}
}

func TestWeekDayString(t *testing.T) {
	if got := weekDayString([]int64{1, 2, 3, 4, 5}); got != "1,2,3,4,5" {
		t.Errorf("weekDayString = %q, want \"1,2,3,4,5\"", got)
	}
}
