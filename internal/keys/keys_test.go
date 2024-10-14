package keys

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// Test HashKey
func TestHashKey(t *testing.T) {
	type testStruct struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	data := testStruct{ID: 1, Name: "Test"}

	expectedBytes, _ := json.Marshal(data)
	expected := string(expectedBytes)

	hash, err := HashKey(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

// Test HashKeyMD5
func TestHashKeyMD5(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
		"num": 123,
	}

	expectedBytes, _ := json.Marshal(data)
	hash := md5.New()
	hash.Write(expectedBytes)
	expected := hex.EncodeToString(hash.Sum(nil))

	hashMD5, err := HashKeyMD5(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if hashMD5 != expected {
		t.Errorf("expected %s, got %s", expected, hashMD5)
	}

	if hashMD5 != "8951e7e02935129b90aacc38e241818e" {
		t.Errorf("expected 8951e7e02935129b90aacc38e241818e, got %s", hashMD5)
	}
}
