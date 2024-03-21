package env

import (
	"os"
	"strconv"
)

// GetString returns the env variable for the given key
// and falls back to the given defaultValue if not set
func GetString(key, defaultValue string) string {
	v, ok := os.LookupEnv(key)
	if ok {
		return v
	}
	return defaultValue
}

// GetInt returns the env variable (parsed as integer) for
// the given key and falls back to the given defaultValue if not set
func GetInt(key string, defaultValue int) (int, error) {
	v, ok := os.LookupEnv(key)
	if ok {
		value, err := strconv.Atoi(v)
		if err != nil {
			return defaultValue, err
		}
		return value, nil
	}
	return defaultValue, nil
}

// GetFloat64 returns the env variable (parsed as float64) for
// the given key and falls back to the given defaultValue if not set
func GetFloat64(key string, defaultValue float64) (float64, error) {
	v, ok := os.LookupEnv(key)
	if ok {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return defaultValue, err
		}
		return value, nil
	}
	return defaultValue, nil
}

// GetBool returns the env variable (parsed as bool) for
// the given key and falls back to the given defaultValue if not set
func GetBool(key string, defaultValue bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if ok {
		value, err := strconv.ParseBool(v)
		if err != nil {
			return defaultValue, err
		}
		return value, nil
	}
	return defaultValue, nil
}
