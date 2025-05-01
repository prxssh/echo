package utils

import "fmt"

func ParseString(m map[string]any, key string, required bool) (string, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required key %q", key)
		}
		return "", nil
	}

	s, ok := raw.(string)
	if !ok {
		if required {
			return "", fmt.Errorf("key %q is not a string", key)
		}
		return "", nil
	}

	return s, nil
}

func ParseInt(m map[string]any, key string, required bool) (int64, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required key %q", key)
		}
		return 0, nil
	}

	s, ok := raw.(int64)
	if !ok {
		if required {
			return 0, fmt.Errorf("key %q is not a string", key)
		}
		return 0, nil
	}

	return s, nil
}
