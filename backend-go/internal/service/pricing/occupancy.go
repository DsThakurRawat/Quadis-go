package pricing

const (
	DefaultExtraAdultPercent   = 30.0
	StandardOccupancyPerRoom   = 2
	DefaultChildFreeUnderAge   = 8
	DefaultChildPercent        = 20.0
	DefaultAdultFromAge        = 13
)

type OccupancyPolicy struct {
	ExtraAdultPercent float64
	ChildFreeUnderAge int
	ChildPercent      float64
	AdultFromAge      int
}

type OccupancyInput struct {
	Adults            int
	ChildAges         []int
	RoomsCount        int
	ChildFreeUnderAge *int
	AdultFromAge      *int
}

type ChargeableGuests struct {
	ExtraAdults   int
	ExtraChildren int
}

// ChargeableGuestsFor computes who has to be paid for and at which rate.
// Two adults per room are included.
// Free age: < childFreeUnderAge (default 8)
// Concession age: >= childFreeUnderAge and < adultFromAge (default 8-12, charged at childPercent)
// Adult-like: >= adultFromAge (default 13+, charged as extra adult)
// Included places are filled with the most expensive heads first.
func ChargeableGuestsFor(input OccupancyInput) ChargeableGuests {
	freeUnder := DefaultChildFreeUnderAge
	if input.ChildFreeUnderAge != nil && *input.ChildFreeUnderAge >= 0 {
		freeUnder = *input.ChildFreeUnderAge
	}

	adultFrom := DefaultAdultFromAge
	if input.AdultFromAge != nil && *input.AdultFromAge >= 0 {
		adultFrom = *input.AdultFromAge
	}

	rooms := input.RoomsCount
	if rooms <= 0 {
		rooms = 1
	}

	var childBandCount int
	var olderChildrenAsAdults int
	for _, age := range input.ChildAges {
		if age >= freeUnder && age < adultFrom {
			childBandCount++
		} else if age >= adultFrom {
			olderChildrenAsAdults++
		}
	}

	adultLike := input.Adults + olderChildrenAsAdults
	included := StandardOccupancyPerRoom * rooms

	extraAdults := adultLike - included
	if extraAdults < 0 {
		extraAdults = 0
	}

	includedLeft := included - adultLike
	if includedLeft < 0 {
		includedLeft = 0
	}

	extraChildren := childBandCount - includedLeft
	if extraChildren < 0 {
		extraChildren = 0
	}

	return ChargeableGuests{
		ExtraAdults:   extraAdults,
		ExtraChildren: extraChildren,
	}
}

func ExtraAdultsFor(input OccupancyInput) int {
	return ChargeableGuestsFor(input).ExtraAdults
}
