package pkg

import (
	"encoding/json"
	"fmt"
	"time"
)

func ConfigToMap[T any](data *T) (map[string]any, error) {
	jsonString, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var s map[string]any
	err = json.Unmarshal(jsonString, &s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func ParseConfig[T any](data map[string]any) (*T, error) {
	jsonString, err := json.Marshal(Convert(data))
	if err != nil {
		return nil, err
	}

	var s T
	err = json.Unmarshal(jsonString, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

type Duration struct {
	time.Duration
}

func (duration *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, duration.Duration.String())), nil
}

func (duration *Duration) UnmarshalJSON(b []byte) error {
	var unmarshalledJson interface{}

	err := json.Unmarshal(b, &unmarshalledJson)
	if err != nil {
		return err
	}

	switch value := unmarshalledJson.(type) {
	case float64:
		duration.Duration = time.Duration(value)
	case string:
		duration.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJson)
	}

	return nil
}

func Convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[string]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k] = Convert(v)
		}
		return m2
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = Convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = Convert(v)
		}
	}
	return i
}
