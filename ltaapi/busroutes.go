package ltaapi

type BusRoute struct {
	ServiceNumber   string  `json:"ServiceNo"`
	Operator        string  `json:"Operator"`
	Direction       int     `json:"Direction"`
	StopSequence    int     `json:"StopSequence"`
	BusStopCode     string  `json:"BusStopCode"`
	Distance        float64 `json:"Distance"`
	WeekDayFirstBus string  `json:"WD_FirstBus"`
	WeekDayLastBus  string  `json:"WD_LastBus"`
	SATFirstBus     string  `json:"SAT_FirstBus"`
	SATLastBus      string  `json:"SAT_LastBus"`
	SUNFirstBus     string  `json:"SUN_FirstBus"`
	SUNLastBus      string  `json:"SUN_LastBus"`
}
