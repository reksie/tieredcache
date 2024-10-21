package keys

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// A wrapper around the JSON serialization function for simplicity
func HashKeyJson(data ...any) (string, error) {
	// Marshal the data directly to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// We can MD5 sum it to "compress" the data into a fixed-size string
func HashKeyMD5(data ...any) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	_, err = hash.Write(jsonBytes)
	if err != nil {
		return "", err
	}

	hashString := hex.EncodeToString(hash.Sum(nil))
	return hashString, nil
}

// HashStructMD5 takes any interface (struct or array), serializes it, and returns a stable MD5 hash.
func HashStructMD5(data interface{}) (string, error) {
	// Serialize the struct/array into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Create a new MD5 hash
	hash := md5.New()
	_, err = hash.Write(jsonData)
	if err != nil {
		return "", err
	}

	// Convert the hash to a hex string
	hashString := hex.EncodeToString(hash.Sum(nil))
	return hashString, nil
}

// HashStructMD5SortedKeys takes a struct/array, sorts the JSON keys, and returns a stable MD5 hash.
func HashStructMD5SortedKeys(data interface{}) (string, error) {
	// Serialize the struct/array into JSON with sorted keys
	sortedJsonData, err := marshalWithSortedKeys(data)
	if err != nil {
		return "", err
	}

	// Create an MD5 hash from the sorted JSON data

	hash := md5.New()
	_, err = hash.Write(sortedJsonData)
	if err != nil {
		return "", err
	}

	// Convert the hash to a hex string
	hashString := hex.EncodeToString(hash.Sum(nil))
	return hashString, nil
}

// Some more complex hashing functions

// sortKeys ensures that the keys of maps are sorted in JSON serialization.
func sortKeys(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Create a new map with sorted keys
		sortedMap := make(map[string]interface{})
		keys := make([]string, 0, len(v))

		// Collect all keys
		for k := range v {
			keys = append(keys, k)
		}

		// Sort keys alphabetically
		sort.Strings(keys)

		// Recursively sort the values
		for _, k := range keys {
			sortedValue, err := sortKeys(v[k])
			if err != nil {
				return nil, err
			}
			sortedMap[k] = sortedValue
		}

		return sortedMap, nil
	case []interface{}:
		// Recursively sort array elements
		sortedArray := make([]interface{}, len(v))
		for i, item := range v {
			sortedItem, err := sortKeys(item)
			if err != nil {
				return nil, err
			}
			sortedArray[i] = sortedItem
		}
		return sortedArray, nil
	default:
		// For other types, return as is
		return data, nil
	}
}

// marshalWithSortedKeys serializes the data with sorted keys for consistent hashing.
func marshalWithSortedKeys(data interface{}) ([]byte, error) {
	// Convert the struct or map to a map[string]interface{} using json.Unmarshal
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var genericData interface{}
	err = json.Unmarshal(jsonBytes, &genericData)
	if err != nil {
		return nil, err
	}

	// Sort the keys in the map or struct
	sortedData, err := sortKeys(genericData)
	if err != nil {
		return nil, err
	}

	// Marshal the sorted data back to JSON
	return json.Marshal(sortedData)
}
