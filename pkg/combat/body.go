package combat

import "reflect"

// ValidBody accepts only a concrete non-nil pointer. Combat identity is the
// body itself, never a display name, prototype or numeric ID shared across kinds.
func ValidBody(body Combatant) bool {
	if body == nil {
		return false
	}
	value := reflect.ValueOf(body)
	return value.Kind() == reflect.Pointer && !value.IsNil() && value.Type().Comparable()
}
