package booking

import "fmt"

const (
	ActDraft     = "DRAFT"
	ActOpen      = "OPEN"
	ActClosed    = "CLOSED"
	ActCancelled = "CANCELLED"

	SlotOpen      = "OPEN"
	SlotLocked    = "LOCKED"
	SlotConfirmed = "CONFIRMED"
	SlotCheckedIn = "CHECKED_IN"
)

type TransitionError struct {
	From, To, Code string
}

func (e TransitionError) Error() string {
	return fmt.Sprintf("illegal transition %s -> %s", e.From, e.To)
}

func CanActivity(from, to string) bool {
	allow := map[string][]string{
		ActDraft:     {ActOpen, ActCancelled},
		ActOpen:      {ActClosed, ActCancelled},
		ActClosed:    {},
		ActCancelled: {},
	}
	return contains(allow[from], to)
}

func CanSlot(from, to, reason string) error {
	ok := false
	switch from + "->" + to {
	case "OPEN->LOCKED":
		ok = reason == "CLAIM"
	case "LOCKED->CONFIRMED":
		ok = reason == "CONFIRM"
	case "LOCKED->OPEN":
		ok = reason == "EXPIRE" || reason == "RELEASE"
	case "CONFIRMED->CHECKED_IN":
		ok = reason == "CHECKIN"
	case "CONFIRMED->OPEN":
		ok = reason == "RELEASE"
	case "CHECKED_IN->CHECKED_IN":
		ok = reason == "CHECKIN"
	}
	if !ok {
		return TransitionError{From: from, To: to, Code: "SLOT_STATE"}
	}
	return nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
