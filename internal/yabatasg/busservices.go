package yabatasg

type BusService struct {
	ServiceNumber    string
	Operator         string
	Direction        int
	Category         string
	OriginCode       string
	DestinationCode  string
	AMPeakFreqMin    int
	AMPeakFreqMax    int
	AMOffpeakFreqMin int
	AMOffpeakFreqMax int
	PMPeakFreqMin    int
	PMPeakFreqMax    int
	PMOffpeakFreqMin int
	PMOffpeakFreqMax int
	LoopDesc         string
}
