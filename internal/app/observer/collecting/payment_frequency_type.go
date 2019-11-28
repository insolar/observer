package collecting

// Status type of swap
type PaymentFrequency int

const (
	FrequencyUnknown PaymentFrequency = iota
	FrequencyHalfHour
	FrequencyWeek
	FrequencyMonth
	FrequencyYear
)

func (s *PaymentFrequency) String() string {
	switch *s {
	case FrequencyHalfHour:
		return "half-hour"
	case FrequencyWeek:
		return "week"
	case FrequencyMonth:
		return "month"
	case FrequencyYear:
		return "year"
	default:
		return "unknown"
	}
}
