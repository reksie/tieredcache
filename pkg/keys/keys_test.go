package keys

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// Test HashKey with multiple arguments
func TestHashKey(t *testing.T) {
	// Test with multiple data types
	data := []interface{}{"string", 123, "another", []string{"one", "two", "three"}}

	// The expected JSON result would be an array of the inputs
	expectedBytes, _ := json.Marshal(data)
	expected := string(expectedBytes)

	// Test the HashKey function
	hash, err := HashKey("string", 123, "another", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Compare the hash result with the expected value
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestHashList(t *testing.T) {
	// Test with multiple data types
	data := []interface{}{"string", 123, "another", []string{"one", "two", "three"}}

	// The expected JSON result would be an array of the inputs
	expectedBytes, _ := json.Marshal(data)
	expected := string(expectedBytes)

	// Test the HashKey function
	hash, err := HashKey("string", 123, "another", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Compare the hash result with the expected value
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

// Test HashKeyMD5 with multiple arguments
func TestHashKeyMD5(t *testing.T) {
	// Test with multiple data types
	data := []interface{}{"string", 123, "another", []string{"one", "two", "three"}}

	// The expected MD5 hash is based on the JSON-encoded array
	expectedBytes, _ := json.Marshal(data)
	hash := md5.New()
	hash.Write(expectedBytes)
	expected := hex.EncodeToString(hash.Sum(nil))

	// Test the HashKeyMD5 function
	hashMD5, err := HashKeyMD5("string", 123, "another", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Compare the hash result with the expected MD5 hash
	if hashMD5 != expected {
		t.Errorf("expected %s, got %s", expected, hashMD5)
	}
}
