package signal

import (
	"fmt"
	"strings"

	"studio-engine/internal/apperr"
)

// EnumChoice is one value of an attribute's EnumType: its label and ord.
type EnumChoice struct {
	Label string
	Ord   int64
}

// Rules a live write must also pass before it reaches the wire (change 0186). Validate keeps
// the frozen Python engine's checks and texts; these add what the attribute's type allows.

// checkEnum refuses an ord outside the attribute's EnumType (TMODE-10). Without a known
// EnumType only Validate's integer range applies.
func checkEnum(item Signal, value any) error {
	if item.Kind != "Enum" || len(item.Enums) == 0 {
		return nil
	}
	number := toBig(value).Int64()
	choices := make([]string, len(item.Enums))
	for i, choice := range item.Enums {
		if choice.Ord == number {
			return nil
		}
		choices[i] = fmt.Sprintf("%d (%s)", choice.Ord, choice.Label)
	}
	return apperr.ValueError(fmt.Sprintf("%d is not a choice of %s. Use one of: %s", number, item.Ref,
		strings.Join(choices, ", ")))
}

// checkVisible refuses text a VisString cannot carry: it is ISO 646 (printable ASCII,
// space to ~), so other characters would reach the wire as UTF-8 bytes no client expects.
func checkVisible(item Signal, value any) error {
	text, ok := value.(string)
	if !ok || !strings.HasPrefix(item.Kind, "VisString") {
		return nil
	}
	for _, character := range text {
		if character < ' ' || character > '~' {
			return apperr.ValueError(fmt.Sprintf("VisString text takes plain ASCII characters only (letters, "+
				"digits, space and ! to ~); remove %q", string(character)))
		}
	}
	return nil
}

// checkQuality validates a quality given with a value in quality words, not integer words.
func checkQuality(kind string, quality any) (any, error) {
	checked, err := Validate(kind, quality)
	if err != nil {
		return nil, apperr.ValueError("Quality must be a whole number from 0 to 8191 (the 13 quality bits)")
	}
	return checked, nil
}
