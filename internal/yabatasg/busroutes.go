package yabatasg

import "time"

type BusRoute struct {
	ServiceNumber   string
	Operator        string
	Direction       int
	StopSequence    int
	BusStopCode     string
	Distance        float64
	WeekDayFirstBus time.Time
	WeekDayLastBus  time.Time
	SATFirstBus     time.Time
	SATLastBus      time.Time
	SUNFirstBus     time.Time
	SUNLastBus      time.Time
}
