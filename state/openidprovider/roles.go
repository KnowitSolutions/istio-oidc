package openidprovider

import (
	"fmt"
	"strconv"
)

func extractRoles(path []string, obj interface{}) ([]string, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	
	dict, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected value of type %T, expected map[string]interface{}", obj)
	}

	id := path[0]
	obj, ok = dict[id]
	if !ok {
		return nil, nil
	}

	if len(path) == 1 {
		return convert(obj)
	} else {
		return extractRoles(path[1:], obj)
	}
}

func convert(obj interface{}) ([]string, error) {
	arr, ok := obj.([]interface{})
	if ok {
		return convertSlice(arr)
	}

	str, err := convertValue(obj)
	if err != nil {
		return nil, err
	}
	return []string{str}, nil
}

func convertSlice(arr []interface{}) ([]string, error) {
	strs := make([]string, len(arr))
	for i, el := range arr {
		str, err := convertValue(el)
		if err != nil {
			return nil, err
		}
		strs[i] = str
	}
	return strs, nil
}

func convertValue(obj interface{}) (string, error) {
	switch val := obj.(type) {
	case bool:
		return strconv.FormatBool(val), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case string:
		return val, nil
	case nil:
		return "null", nil
	default:
		return "", fmt.Errorf("unexpected value of type %T", val)
	}
}
