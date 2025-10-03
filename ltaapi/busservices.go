package ltaapi

type BusService struct {
	ServiceNumber   string `json:"ServiceNo"`
	Operator        string `json:"Operator"`
	Direction       int    `json:"Direction"`
	Category        string `json:"Category"`
	OriginCode      string `json:"OriginCode"`
	DestinationCode string `json:"DestinationCode"`
	AMPeakFreq      string `json:"AM_Peak_Freq"`
	AMOffpeakFreq   string `json:"AM_Offpeak_Freq"`
	PMPeakFreq      string `json:"PM_Peak_Freq"`
	PMOffpeakFreq   string `json:"PM_Offpeak_Freq"`
	LoopDesc        string `json:"LoopDesc"`
}
