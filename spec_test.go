package cron

import (
	"testing"
	"time"
)

func TestActivation(t *testing.T) {
	tests := []struct {
		time, spec string
		expected   bool
	}{
		// Every fifteen minutes.
		{"Mon Jul 9 15:00 2012", "0 0/15 * * *", true},
		{"Mon Jul 9 15:45 2012", "0 0/15 * * *", true},
		{"Mon Jul 9 15:40 2012", "0 0/15 * * *", false},

		// Every fifteen minutes, starting at 5 minutes.
		{"Mon Jul 9 15:05 2012", "0 5/15 * * *", true},
		{"Mon Jul 9 15:20 2012", "0 5/15 * * *", true},
		{"Mon Jul 9 15:50 2012", "0 5/15 * * *", true},

		// Named months
		{"Sun Jul 15 15:00 2012", "0 0/15 * * Jul", true},
		{"Sun Jul 15 15:00 2012", "0 0/15 * * Jun", false},

		// Everything set.
		{"Sun Jul 15 08:30 2012", "0 30 08 ? Jul Sun", true},
		{"Sun Jul 15 08:30 2012", "0 30 08 15 Jul ?", true},
		{"Mon Jul 16 08:30 2012", "0 30 08 ? Jul Sun", false},
		{"Mon Jul 16 08:30 2012", "0 30 08 15 Jul ?", false},

		// Predefined schedules
		{"Mon Jul 9 15:00 2012", "@hourly", true},
		{"Mon Jul 9 15:04 2012", "@hourly", false},
		{"Mon Jul 9 15:00 2012", "@daily", false},
		{"Mon Jul 9 00:00 2012", "@daily", true},
		{"Mon Jul 9 00:00 2012", "@weekly", false},
		{"Sun Jul 8 00:00 2012", "@weekly", true},
		{"Sun Jul 8 01:00 2012", "@weekly", false},
		{"Sun Jul 8 00:00 2012", "@monthly", false},
		{"Sun Jul 1 00:00 2012", "@monthly", true},

		// Test interaction of DOW and DOM.
		// If both are specified, then only one needs to match.
		{"Sun Jul 15 00:00 2012", "0 * * 1,15 * Sun", true},
		{"Fri Jun 15 00:00 2012", "0 * * 1,15 * Sun", true},
		{"Wed Aug 1 00:00 2012", "0 * * 1,15 * Sun", true},

		// However, if one has a star, then both need to match.
		{"Sun Jul 15 00:00 2012", "0 * * * * Mon", false},
		{"Sun Jul 15 00:00 2012", "0 * * */10 * Sun", false},
		{"Mon Jul 9 00:00 2012", "0 * * 1,15 * *", false},
		{"Sun Jul 15 00:00 2012", "0 * * 1,15 * *", true},
		{"Sun Jul 15 00:00 2012", "0 * * */2 * Sun", true},
	}

	for _, test := range tests {
		sched, err := Parse(test.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(getTime(test.time).Add(-1 * time.Second))
		expected := getTime(test.time)
		if test.expected && expected != actual || !test.expected && expected == actual {
			t.Errorf("Fail evaluating %s on %s: (expected) %s != %s (actual)",
				test.spec, test.time, expected, actual)
		}
	}
}

func TestNext(t *testing.T) {
	runs := []struct {
		time, spec string
		expected   string
	}{
		// Simple cases
		{"Mon Jul 9 14:45 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},
		{"Mon Jul 9 14:59 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},
		{"Mon Jul 9 14:59:59 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},

		// Wrap around hours
		{"Mon Jul 9 15:45 2012", "0 20-35/15 * * *", "Mon Jul 9 16:20 2012"},

		// Wrap around days
		{"Mon Jul 9 23:46 2012", "0 */15 * * *", "Tue Jul 10 00:00 2012"},
		{"Mon Jul 9 23:45 2012", "0 20-35/15 * * *", "Tue Jul 10 00:20 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * * *", "Tue Jul 10 00:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 1/2 * *", "Tue Jul 10 01:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 10-12 * *", "Tue Jul 10 10:20:15 2012"},

		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 1/2 */2 * *", "Thu Jul 11 01:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * 9-20 * *", "Wed Jul 10 00:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * 9-20 Jul *", "Wed Jul 10 00:20:15 2012"},

		// Wrap around months
		{"Mon Jul 9 23:35 2012", "0 0 0 9 Apr-Oct ?", "Thu Aug 9 00:00 2012"},
		{"Mon Jul 9 23:35 2012", "0 0 0 */5 Apr,Aug,Oct Mon", "Mon Aug 6 00:00 2012"},
		{"Mon Jul 9 23:35 2012", "0 0 0 */5 Oct Mon", "Mon Oct 1 00:00 2012"},

		// Wrap around years
		{"Mon Jul 9 23:35 2012", "0 0 0 * Feb Mon", "Mon Feb 4 00:00 2013"},
		{"Mon Jul 9 23:35 2012", "0 0 0 * Feb Mon/2", "Fri Feb 1 00:00 2013"},

		// Wrap around minute, hour, day, month, and year
		{"Mon Dec 31 23:59:45 2012", "0 * * * * *", "Tue Jan 1 00:00:00 2013"},

		// Leap year
		{"Mon Jul 9 23:35 2012", "0 0 0 29 Feb ?", "Mon Feb 29 00:00 2016"},

		// Daylight savings time 2am EST (-5) -> 3am EDT (-4)
		{"2012-03-11T00:00:00-0500", "0 30 2 11 Mar ?", "2013-03-11T02:30:00-0400"},

		// hourly job
		{"2012-03-11T00:00:00-0500", "0 0 * * * ?", "2012-03-11T01:00:00-0500"},
		{"2012-03-11T01:00:00-0500", "0 0 * * * ?", "2012-03-11T03:00:00-0400"},
		{"2012-03-11T03:00:00-0400", "0 0 * * * ?", "2012-03-11T04:00:00-0400"},
		{"2012-03-11T04:00:00-0400", "0 0 * * * ?", "2012-03-11T05:00:00-0400"},

		// 1am nightly job
		{"2012-03-11T00:00:00-0500", "0 0 1 * * ?", "2012-03-11T01:00:00-0500"},
		{"2012-03-11T01:00:00-0500", "0 0 1 * * ?", "2012-03-12T01:00:00-0400"},

		// 2am nightly job (skipped)
		{"2012-03-11T00:00:00-0500", "0 0 2 * * ?", "2012-03-12T02:00:00-0400"},

		// Daylight savings time 2am EDT (-4) => 1am EST (-5)
		{"2012-11-04T00:00:00-0400", "0 30 2 04 Nov ?", "2012-11-04T02:30:00-0500"},
		{"2012-11-04T01:45:00-0400", "0 30 1 04 Nov ?", "2012-11-04T01:30:00-0500"},

		// hourly job
		{"2012-11-04T00:00:00-0400", "0 0 * * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-04T01:00:00-0400", "0 0 * * * ?", "2012-11-04T01:00:00-0500"},
		{"2012-11-04T01:00:00-0500", "0 0 * * * ?", "2012-11-04T02:00:00-0500"},

		// 1am nightly job (runs twice)
		{"2012-11-04T00:00:00-0400", "0 0 1 * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-04T01:00:00-0400", "0 0 1 * * ?", "2012-11-04T01:00:00-0500"},
		{"2012-11-04T01:00:00-0500", "0 0 1 * * ?", "2012-11-05T01:00:00-0500"},

		// 2am nightly job
		{"2012-11-04T00:00:00-0400", "0 0 2 * * ?", "2012-11-04T02:00:00-0500"},
		{"2012-11-04T02:00:00-0500", "0 0 2 * * ?", "2012-11-05T02:00:00-0500"},

		// 3am nightly job
		{"2012-11-04T00:00:00-0400", "0 0 3 * * ?", "2012-11-04T03:00:00-0500"},
		{"2012-11-04T03:00:00-0500", "0 0 3 * * ?", "2012-11-05T03:00:00-0500"},

		// Unsatisfiable
		{"Mon Jul 9 23:35 2012", "0 0 0 30 Feb ?", ""},
		{"Mon Jul 9 23:35 2012", "0 0 0 31 Apr ?", ""},
	}

	for _, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(getTime(c.time))
		expected := getTime(c.expected)
		if !actual.Equal(expected) {
			t.Errorf("%s, \"%s\": (expected) %v != %v (actual)", c.time, c.spec, expected, actual)
		}
	}
}

func TestErrors(t *testing.T) {
	invalidSpecs := []string{
		"xyz",
		"60 0 * * *",
		"0 60 * * *",
		"0 0 * * XYZ",
	}
	for _, spec := range invalidSpecs {
		_, err := Parse(spec)
		if err == nil {
			t.Error("expected an error parsing: ", spec)
		}
	}
}

func getTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse("Mon Jan 2 15:04 2006", value)
	if err != nil {
		t, err = time.Parse("Mon Jan 2 15:04:05 2006", value)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05-0700", value)
			if err != nil {
				panic(err)
			}
			// Daylight savings time tests require location
			if ny, err := time.LoadLocation("America/New_York"); err == nil {
				t = t.In(ny)
			}
		}
	}

	return t
}

func TestNextWithTz(t *testing.T) {
	runs := []struct {
		time, spec string
		expected   string
	}{
		// Failing tests
		{"2016-01-03T13:09:03+0530", "0 14 14 * * *", "2016-01-03T14:14:00+0530"},
		{"2016-01-03T04:09:03+0530", "0 14 14 * * ?", "2016-01-03T14:14:00+0530"},

		// Passing tests
		{"2016-01-03T14:09:03+0530", "0 14 14 * * *", "2016-01-03T14:14:00+0530"},
		{"2016-01-03T14:00:00+0530", "0 14 14 * * ?", "2016-01-03T14:14:00+0530"},
	}
	for _, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(getTimeTZ(c.time))
		expected := getTimeTZ(c.expected)
		if !actual.Equal(expected) {
			t.Errorf("%s, \"%s\": (expected) %v != %v (actual)", c.time, c.spec, expected, actual)
		}
	}
}

func getTimeTZ(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse("Mon Jan 2 15:04 2006", value)
	if err != nil {
		t, err = time.Parse("Mon Jan 2 15:04:05 2006", value)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05-0700", value)
			if err != nil {
				panic(err)
			}
		}
	}

	return t
}

func TestPrevMatching(t *testing.T) {
	runs := []struct {
		time, spec string
		expected   string
	}{
		// Simple cases
		{"Mon Jul 9 14:45 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},
		{"Mon Jul 9 14:59 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},
		{"Mon Jul 9 14:59:59 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},

		// Wrap around hours
		{"Mon Jul 9 15:45 2012", "0 20-35/15 * * *", "Mon Jul 9 16:20 2012"},

		// Wrap around days
		{"Mon Jul 9 23:46 2012", "0 */15 * * *", "Tue Jul 10 00:00 2012"},
		{"Mon Jul 9 23:45 2012", "0 20-35/15 * * *", "Tue Jul 10 00:20 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * * *", "Tue Jul 10 00:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 1/2 * *", "Tue Jul 10 21:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 10-12 * *", "Tue Jul 10 10:20:15 2012"},

		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 1/2 */2 * *", "Thu Jul 11 01:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * 9-20 * *", "Wed Jul 10 00:20:15 2012"},
		{"Mon Jul 9 23:35:51 2012", "15/35 20-35/15 * 9-20 Jul *", "Wed Jul 10 00:20:15 2012"},

		// Wrap around months
		{"Mon Jul 9 23:35 2012", "0 0 0 9 Apr-Oct ?", "Thu Aug 9 00:00 2012"},
		{"Mon Jul 9 23:35 2012", "0 0 0 */5 Apr,Aug,Oct Mon", "Mon Aug 6 00:00 2012"},
		{"Mon Jul 9 23:35 2012", "0 0 0 */5 Oct Mon", "Mon Oct 1 00:00 2012"},

		// Wrap around years
		{"Mon Jul 9 23:35 2012", "0 0 0 * Feb Mon", "Mon Feb 4 00:00 2013"},
		{"Mon Jul 9 23:35 2012", "0 0 0 * Feb Mon/2", "Fri Feb 1 00:00 2013"},

		// Wrap around minute, hour, day, month, and year
		{"Mon Dec 31 23:59:45 2012", "0 * * * * *", "Tue Jan 1 00:00:00 2013"},

		// Leap year
		{"Mon Jul 9 23:35 2012", "0 0 0 29 Feb ?", "Mon Feb 29 00:00 2016"},

		// Daylight savings time 2am EST (-5) -> 3am EDT (-4)
		{"2012-03-11T00:00:00-0500", "0 30 2 11 Mar ?", "2013-03-11T02:30:00-0400"},

		// hourly job
		{"2012-03-11T00:00:00-0500", "0 0 * * * ?", "2012-03-11T01:00:00-0500"},
		{"2012-03-11T01:00:00-0500", "0 0 * * * ?", "2012-03-11T03:00:00-0400"},
		{"2012-03-11T03:00:00-0400", "0 0 * * * ?", "2012-03-11T04:00:00-0400"},
		{"2012-03-11T04:00:00-0400", "0 0 * * * ?", "2012-03-11T05:00:00-0400"},

		// 1am nightly job
		{"2012-03-11T00:00:00-0500", "0 0 1 * * ?", "2012-03-11T01:00:00-0500"},
		{"2012-03-11T01:00:00-0500", "0 0 1 * * ?", "2012-03-12T01:00:00-0400"},

		// 2am nightly job (skipped)
		{"2012-03-11T00:00:00-0500", "0 0 2 * * ?", "2012-03-12T02:00:00-0400"},

		// Daylight savings time 2am EDT (-4) => 1am EST (-5)
		{"2012-11-04T00:00:00-0400", "0 30 2 04 Nov ?", "2012-11-04T02:30:00-0500"},
		{"2012-11-04T01:45:00-0400", "0 30 1 04 Nov ?", "2012-11-04T01:30:00-0500"},

		// hourly job
		{"2012-11-04T00:00:00-0400", "0 0 * * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-04T01:00:00-0400", "0 0 * * * ?", "2012-11-04T01:00:00-0500"},
		{"2012-11-04T01:00:00-0500", "0 0 * * * ?", "2012-11-04T02:00:00-0500"},

		// 1am nightly job (runs twice)
		{"2012-11-04T00:00:00-0400", "0 0 1 * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-04T01:00:00-0400", "0 0 1 * * ?", "2012-11-04T01:00:00-0500"},
		{"2012-11-04T01:00:00-0500", "0 0 1 * * ?", "2012-11-05T01:00:00-0500"},

		// 2am nightly job
		{"2012-11-04T00:00:00-0400", "0 0 2 * * ?", "2012-11-04T02:00:00-0500"},
		{"2012-11-04T02:00:00-0500", "0 0 2 * * ?", "2012-11-05T02:00:00-0500"},

		// 3am nightly job
		{"2012-11-04T00:00:00-0400", "0 0 3 * * ?", "2012-11-04T03:00:00-0500"},
		{"2012-11-04T03:00:00-0500", "0 0 3 * * ?", "2012-11-05T03:00:00-0500"},

		// Unsatisfiable
		{"Mon Jul 9 23:35 2012", "0 0 0 30 Feb ?", ""},
		{"Mon Jul 9 23:35 2012", "0 0 0 31 Apr ?", ""},
	}

	const probeCount = 15
	type probe struct {
		now      time.Time
		nextTime time.Time
	}

	for _, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}

		timeProbe := make([]probe, 0, probeCount)

		now := getTime(c.time)
		for i := 0; i < probeCount; i++ {
			nextTime := sched.Next(now)
			timeProbe = append(timeProbe, probe{
				now:      now,
				nextTime: nextTime,
			})
			now = nextTime.Add(1 * time.Second)
		}

		for i := probeCount - 1; i >= 1; i-- {
			thisStep := timeProbe[i]
			prevStep := timeProbe[i-1]

			prevTime := sched.Prev(thisStep.nextTime)

			if !prevTime.Equal(prevStep.nextTime) {
				t.Errorf("%s, \"%s\":", c.time, c.spec)
				t.Errorf("FAIL(%d) (expected) %v -> %v == %v (actual)", i, thisStep.nextTime, prevStep.nextTime, prevTime)
				t.Errorf("time probes: ")
				for _, v := range timeProbe {
					t.Errorf("-> %v", v.nextTime)
				}
				return
			}
		}
	}
}

func TestPrev(t *testing.T) {
	runs := []struct {
		time, spec string
		expected   string
	}{
		// Simple cases
		{"Mon Jul 9 14:45 2012", "0 0/15 * * *", "Mon Jul 9 14:30 2012"},
		{"Mon Jul 9 14:59 2012", "0 0/15 * * *", "Mon Jul 9 14:45 2012"},
		{"Mon Jul 9 14:59:59 2012", "0 0/15 * * *", "Mon Jul 9 14:45 2012"},

		{"Mon Jul 9 15:15 2012", "0 0/15 * * *", "Mon Jul 9 15:00 2012"},
		{"Mon Jul 9 15:15:59 2012", "0 0/15 * * *", "Mon Jul 9 15:15 2012"},
		{"Mon Jul 9 14:01 2012", "0 0/15 * * *", "Mon Jul 9 14:00 2012"},
		{"Mon Jul 9 14:00:59 2012", "0 0/15 * * *", "Mon Jul 9 14:00 2012"},

		// Wrap around hours
		{"Mon Jul 9 15:05 2012", "0 20-35/15 * * *", "Mon Jul 9 14:35 2012"},

		// Wrap around days
		{"Mon Jul 9 00:01 2012", "0 */15 * * *", "Tue Jul 09 00:00 2012"},
		{"Mon Jul 9 00:15 2012", "0 */15 * * *", "Tue Jul 09 00:00 2012"},
		{"Mon Jul 9 00:15 2012", "0 20-35/15 * * *", "Tue Jul 08 23:35 2012"},
		{"Mon Jul 9 00:05:51 2012", "15/35 20-35/15 * * *", "Tue Jul 08 23:35:50 2012"},
		{"Mon Jul 9 01:05:52 2012", "15/35 20-35/15 1/2 * *", "Tue Jul 08 23:35:50 2012"},
		{"Mon Jul 9 00:05:53 2012", "15/35 20-35/15 10-12 * *", "Tue Jul 08 12:35:50 2012"},

		{"Mon Jul 9 00:05:51 2012", "15/35 20-35/15 1/2 */2 * *", "Thu Jul 07 23:35:50 2012"},
		{"Mon Jul 9 00:05:52 2012", "15/35 20-35/15 * 9-20 * *", "Wed Jun 20 23:35:50 2012"},
		{"Mon Jul 21 00:05:53 2012", "15/35 20-35/15 * 9-20 Jul *", "Wed Jul 20 23:35:50 2012"},

		// Wrap around months
		{"Mon Jul 9 23:35 2012", "0 0 0 9 Apr-Oct ?", "Thu Jul 9 00:00 2012"},
		{"Mon Jul 9 23:35 2012", "0 0 0 */5 Apr,Aug,Oct Mon", "Mon Apr 16 00:00 2012"},
		{"Mon Dec 9 23:35 2012", "0 0 0 */5 Oct Mon", "Mon Oct 1 00:00 2012"},

		// Wrap around years
		{"Mon Jan 9 23:35 2013", "0 0 0 * Feb Mon", "Mon Feb 27 00:00 2012"},
		{"Mon Jan 9 23:35 2013", "0 0 0 * Feb Mon/2", "Fri Feb 29 00:00 2012"},

		// Wrap around minute, hour, day, month, and year
		{"Tue Jan 1 00:00:00 2013", "0 * * * * *", "Mon Dec 31 23:59:00 2012"},

		// Leap year
		{"Mon Jul 9 23:35 2013", "0 0 0 29 Feb ?", "Mon Feb 29 00:00 2012"},

		// Daylight savings time 2am EST (-5) -> 3am EDT (-4)
		{"2013-03-11T03:30:00-0400", "0 30 2 11 Mar ?", "2013-03-11T02:30:00-0400"},
		{"2012-03-11T03:30:00-0400", "0 30 2 11 Mar ?", "2011-03-11T02:30:00-0500"},

		// hourly job
		{"2012-03-11T00:00:00-0500", "0 0 * * * ?", "2012-03-10T23:00:00-0500"},
		{"2012-03-11T01:00:00-0500", "0 0 * * * ?", "2012-03-11T01:00:00-0400"},
		{"2012-03-11T03:00:00-0400", "0 0 * * * ?", "2012-03-11T02:00:00-0400"},
		{"2012-03-11T04:00:00-0400", "0 0 * * * ?", "2012-03-11T03:00:00-0400"},

		{"2013-03-11T00:01:00-0500", "0 0 * * * ?", "2013-03-11T01:00:00-0400"},
		{"2013-03-11T01:01:00-0500", "0 0 * * * ?", "2013-03-11T02:00:00-0400"},
		{"2013-03-11T03:01:00-0400", "0 0 * * * ?", "2013-03-11T03:00:00-0400"},
		{"2013-03-11T04:01:00-0400", "0 0 * * * ?", "2013-03-11T04:00:00-0400"},

		// 1am nightly job
		{"2012-03-11T00:00:00-0500", "0 0 1 * * ?", "2012-03-10T01:00:00-0500"},
		{"2012-03-12T01:00:00-0400", "0 0 1 * * ?", "2012-03-11T01:00:00-0500"},

		// 2am nightly job (skipped)
		{"2012-03-12T02:00:00-0400", "0 0 2 * * ?", "2012-03-10T02:00:00-0500"}, // 2012-03-11 02:00:00 must be skipped

		// Daylight savings time 2am EDT (-4) => 1am EST (-5)
		{"2012-11-05T02:30:00-0500", "0 30 2 04 Nov ?", "2012-11-04T02:30:00-0500"},
		{"2012-11-05T01:30:00-0500", "0 30 1 04 Nov ?", "2012-11-04T01:30:00-0500"},

		// hourly job
		{"2012-11-04T01:00:00-0400", "0 0 * * * ?", "2012-11-04T00:00:00-0400"},
		{"2012-11-04T01:00:00-0500", "0 0 * * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-04T02:00:00-0500", "0 0 * * * ?", "2012-11-04T01:00:00-0500"},

		// 1am nightly job (runs twice)
		{"2012-11-04T01:00:00-0400", "0 0 1 * * ?", "2012-11-03T01:00:00-0400"},
		{"2012-11-04T01:00:00-0500", "0 0 1 * * ?", "2012-11-04T01:00:00-0400"},
		{"2012-11-05T01:00:00-0500", "0 0 1 * * ?", "2012-11-04T01:00:00-0500"},

		// 2am nightly job
		{"2012-11-04T02:00:00-0500", "0 0 2 * * ?", "2012-11-03T02:00:00-0400"},
		{"2012-11-05T02:00:00-0500", "0 0 2 * * ?", "2012-11-04T02:00:00-0500"},

		// 3am nightly job
		{"2012-11-04T00:00:00-0400", "0 0 3 * * ?", "2012-11-03T03:00:00-0400"},
		{"2012-11-04T03:00:00-0500", "0 0 3 * * ?", "2012-11-03T03:00:00-0400"},

		// Unsatisfiable
		{"Mon Jul 9 23:35 2012", "0 0 0 30 Feb ?", ""},
		{"Mon Jul 9 23:35 2012", "0 0 0 31 Apr ?", ""},
	}

	for i, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}

		actual := sched.Prev(getTime(c.time))
		expected := getTime(c.expected)
		if !actual.Equal(expected) {
			t.Fatalf("#%d %s, \"%s\": (expected) %v != %v (actual)", i, c.time, c.spec, expected, actual)
		}
	}
}
