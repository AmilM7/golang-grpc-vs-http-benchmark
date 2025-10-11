package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"google.golang.org/grpc"
	userpb "golang-grpc/pkg/gen/user/v1"
)

const (
	httpURL     = "http://127.0.0.1:8087/users"
	grpcAddress = "127.0.0.1:50055"
	iterations  = 1000
)

func main() {
	fmt.Println("=== Benchmark: HTTP vs gRPC ===")
	rand.Seed(time.Now().UnixNano())

	httpStats := measureHTTPBatch()
	grpcStats := measureGRPCBatch()

	fmt.Println()
	fmt.Printf("HTTP: avg=%v | min=%v | max=%v\n", httpStats.Avg, httpStats.Min, httpStats.Max)
	fmt.Printf("gRPC: avg=%v | min=%v | max=%v\n", grpcStats.Avg, grpcStats.Min, grpcStats.Max)
}

type stats struct {
	Avg, Min, Max time.Duration
}

func measureHTTPBatch() stats {
	var total time.Duration
	min := time.Hour
	max := time.Duration(0)

	for i := 0; i < iterations; i++ {
		name := fmt.Sprintf("User-%d", i)
		email := fmt.Sprintf("user%d@example.com", i)
		payload := map[string]string{"name": name, "email": email}
		body, _ := json.Marshal(payload)

		start := time.Now()
		resp, err := http.Post(httpURL, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Fatalf("HTTP request failed: %v", err)
		}
		resp.Body.Close()
		elapsed := time.Since(start)

		total += elapsed
		if elapsed < min {
			min = elapsed
		}
		if elapsed > max {
			max = elapsed
		}
	}

	return stats{
		Avg: total / iterations,
		Min: min,
		Max: max,
	}
}

func measureGRPCBatch() stats {
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("gRPC connection failed: %v", err)
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)

	var total time.Duration
	min := time.Hour
	max := time.Duration(0)

	for i := 0; i < iterations; i++ {
		req := &userpb.CreateUserRequest{
			Name:  fmt.Sprintf("GrpcUser-%d", i),
			Email: fmt.Sprintf("grpc%d@example.com", i),
		}

		start := time.Now()
		_, err := client.CreateUser(context.Background(), req)
		if err != nil {
			log.Fatalf("gRPC call failed: %v", err)
		}
		elapsed := time.Since(start)

		total += elapsed
		if elapsed < min {
			min = elapsed
		}
		if elapsed > max {
			max = elapsed
		}
	}

	return stats{
		Avg: total / iterations,
		Min: min,
		Max: max,
	}
}
