package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/reksie/memocache/pkg/keys"
)

func main() {
	type Nest struct {
		SubKeyB int
		SubKeyA int
	}
	// Example struct and map with unordered keys

	type Example struct {
		Name string
		Meta map[string]interface{}
		Nest Nest
	}

	exampleData := Example{
		Name: "test",
		Meta: map[string]interface{}{
			"keyB": "value2",
			"keyA": "value1",
			"keyC": map[string]interface{}{
				"subKeyB": 2,
				"subKeyA": 1,
			},
		},
		Nest: Nest{
			SubKeyA: 1,
			SubKeyB: 2,
		},
	}

	// Hash the struct with sorted keys using MD5
	hash, err := keys.HashStructMD5SortedKeys(exampleData)
	if err != nil {
		log.Fatalf("Failed to hash struct: %v", err)
	}

	fmt.Println("MD5 Hash with sorted keys:", hash)

	type Another struct {
		Name string
		Meta map[string]interface{}
		Nest Nest
	}
	moreData := Another{
		Name: "test",
		Nest: Nest{
			SubKeyA: 1,
			SubKeyB: 2,
		},
		Meta: map[string]interface{}{
			"keyA": "value1",
			"keyB": "value2",
			"keyC": map[string]interface{}{
				"subKeyA": 1,
				"subKeyB": 2,
			},
		},
	}

	// Hash the struct with sorted keys using MD5
	hash, err = keys.HashStructMD5SortedKeys(moreData)
	if err != nil {
		log.Fatalf("Failed to hash struct: %v", err)
	}

	fmt.Println("MD5 Hash with sorted keys:", hash)

	// without sorted
	hash, err = keys.HashStructMD5(exampleData)
	if err != nil {
		log.Fatalf("Failed to hash struct: %v", err)
	}
	fmt.Printf("MD5 Hash without sorted keys: %s\n", hash)

	type Simple struct {
		Name string
		Age  int
		Tags []string
	}

	simpleData := Simple{
		Tags: []string{"tag1", "tag2"},
		Name: "test",
		Age:  30,
	}

	// Hash the struct with sorted keys using MD5
	hash, err = keys.HashStructMD5(simpleData)
	if err != nil {
		log.Fatalf("Failed to hash struct: %v", err)
	}

	fmt.Println("MD5 with out map:", hash)

	simpleData2 := Simple{
		Age:  30,
		Name: "test",
		Tags: []string{"tag1", "tag2"},
	}

	hash, err = keys.HashStructMD5(simpleData2)
	if err != nil {
		log.Fatalf("Failed to hash struct: %v", err)
	}

	fmt.Println("MD5 with out map:", hash)

	jsonBytes, err := json.Marshal(simpleData)
	if err != nil {
		log.Fatalf("Failed to marshal struct: %v", err)
	}

	fmt.Println(string(jsonBytes))

	jsonBytes, err = json.Marshal(simpleData2)
	if err != nil {
		log.Fatalf("Failed to marshal struct: %v", err)
	}

	fmt.Println(string(jsonBytes))

	const useMd5 = true
	var hashKey func(data ...any) (string, error)
	if useMd5 {
		hashKey = keys.HashKeyMD5
	} else {
		hashKey = keys.HashKey
	}

	key, err := HashKeyJson("string", 123, "another", []string{"one", "two", "three"}, simpleData)
	if err != nil {
		log.Fatalf("Failed to hash key: %v", err)
	}
	fmt.Println(key)
}
