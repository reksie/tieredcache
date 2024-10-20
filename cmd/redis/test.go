package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	type RedisItem struct {
		Value     string `json:"value"`
		CrazyDays int    `json:"crazyDays"`
	}

	// Create and store the RedisItem
	redisItem := RedisItem{
		Value: "this is something else auto marshaling",
	}

	ctx := context.Background()

	// Convert RedisItem to JSON before storing
	jsonData, err := json.Marshal(redisItem)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return
	}

	// Store JSON data in Redis
	err = client.Set(ctx, "key", jsonData, redis.KeepTTL).Err()
	if err != nil {
		fmt.Println("Error setting value:", err)
		return
	}

	// Retrieve the value from Redis
	val, err := client.Get(ctx, "key").Bytes()
	if err != nil {
		fmt.Println("Error getting value:", err)
		return
	}

	// Unmarshal the JSON data back into a RedisItem struct
	var retrievedItem RedisItem
	err = json.Unmarshal(val, &retrievedItem)
	if err != nil {
		fmt.Println("Error unmarshaling:", err)
		return
	}

	fmt.Printf("Retrieved item: %+v\n", retrievedItem.Value)

	brokenVal, err := client.Get(ctx, "invalid").Bytes()
	if err != nil {
		fmt.Println("Error getting value:", err)
		return
	}

	// Unmarshal the JSON data back into a RedisItem struct

	err = json.Unmarshal(brokenVal, &retrievedItem)
	if err != nil {
		fmt.Println("Error unmarshaling:", err)
		return
	}

}
