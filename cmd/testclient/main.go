package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userpb "golang-grpc/pkg/gen/user/v1"
)

const (
	defaultHTTPBaseURL  = "http://127.0.0.1:8087"
	defaultGRPCAddress  = "127.0.0.1:50055"
	defaultIterations   = 100
	defaultConcurrency  = 5
	defaultWarmup       = 20
	defaultRPCTimeoutMs = 2000

	payloadBioRepeat   = 64
	payloadAvatarBytes = 4096
	payloadTagCount    = 6
	createDataSalt     = 13
	updateDataSalt     = 101
	addressDataSalt    = 29
	httpEmailDomain    = "example.com"
	grpcEmailDomain    = "rpc.example"
)

var operationsOrder = []string{"create", "update", "get", "delete"}

type benchConfig struct {
	HTTPBaseURL string
	GRPCAddress string
	Iterations  int
	Concurrency int
	Warmup      int
	RPCTimeout  time.Duration
}

type stats struct {
	Avg time.Duration
	Min time.Duration
	Max time.Duration
}

type accumulator struct {
	total time.Duration
	min   time.Duration
	max   time.Duration
	count int
}

func (a *accumulator) add(d time.Duration) {
	a.total += d
	if a.count == 0 || d < a.min {
		a.min = d
	}
	if a.count == 0 || d > a.max {
		a.max = d
	}
	a.count++
}

func (a *accumulator) stats() stats {
	if a.count == 0 {
		return stats{}
	}
	return stats{
		Avg: time.Duration(int64(a.total) / int64(a.count)),
		Min: a.min,
		Max: a.max,
	}
}

type statCollector map[string]*accumulator

func newCollector(keys ...string) statCollector {
	c := make(statCollector, len(keys))
	for _, key := range keys {
		c[key] = &accumulator{}
	}
	return c
}

func (c statCollector) add(key string, d time.Duration) {
	if acc, ok := c[key]; ok {
		acc.add(d)
	}
}

func (c statCollector) stats() map[string]stats {
	out := make(map[string]stats, len(c))
	for key, acc := range c {
		out[key] = acc.stats()
	}
	return out
}

type wireUser struct {
	ID      string   `json:"id,omitempty"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Phone   string   `json:"phone"`
	Address string   `json:"address"`
	Bio     string   `json:"bio"`
	Tags    []string `json:"tags"`
	Avatar  []byte   `json:"avatar"`
}

func (w wireUser) toCreateRequest() *userpb.CreateUserRequest {
	return &userpb.CreateUserRequest{
		Name:    w.Name,
		Email:   w.Email,
		Phone:   w.Phone,
		Address: w.Address,
		Bio:     w.Bio,
		Tags:    append([]string(nil), w.Tags...),
		Avatar:  append([]byte(nil), w.Avatar...),
	}
}

func (w wireUser) toUpdateRequest(id string) *userpb.UpdateUserRequest {
	return &userpb.UpdateUserRequest{
		Id:      id,
		Name:    w.Name,
		Email:   w.Email,
		Phone:   w.Phone,
		Address: w.Address,
		Bio:     w.Bio,
		Tags:    append([]string(nil), w.Tags...),
		Avatar:  append([]byte(nil), w.Avatar...),
	}
}

func makeUserPayload(prefix, domain string, idx, salt int) wireUser {
	return wireUser{
		Name:    fmt.Sprintf("%s-%d-%d", prefix, idx, salt),
		Email:   fmt.Sprintf("%s%d+%d@%s", prefix, idx, salt, domain),
		Phone:   fmt.Sprintf("+1-800-%04d-%04d", (idx+salt)%10000, (idx*salt+addressDataSalt)%10000),
		Address: fmt.Sprintf("%d %s Benchmark Blvd Suite %d", idx+salt+addressDataSalt, strings.ToUpper(prefix), (idx*salt)%500+1),
		Bio:     buildBio(prefix, idx, salt),
		Tags:    buildTags(prefix, idx, salt),
		Avatar:  buildAvatar(prefix, idx, salt),
	}
}

func buildBio(prefix string, idx, salt int) string {
	fragments := []string{
		"lorem ipsum dolor sit amet",
		"transport benchmark payload",
		"gin vs grpc serialization",
		"protobuf binary framing",
		"contention under load",
	}
	var b strings.Builder
	snippet := fmt.Sprintf("%s user %d salt %d ", prefix, idx, salt)
	for i := 0; i < payloadBioRepeat; i++ {
		b.WriteString(snippet)
		b.WriteString(fragments[(idx+salt+i)%len(fragments)])
		b.WriteByte(' ')
	}
	return b.String()
}

func buildTags(prefix string, idx, salt int) []string {
	tags := make([]string, payloadTagCount)
	for i := range tags {
		tags[i] = fmt.Sprintf("%s-tag-%02d-%d", prefix, i, (idx+salt+i)%500)
	}
	return tags
}

func buildAvatar(prefix string, idx, salt int) []byte {
	data := make([]byte, payloadAvatarBytes)
	base := byte(len(prefix) + idx + salt + addressDataSalt)
	for i := range data {
		data[i] = base + byte((i*13+salt)%251)
	}
	return data
}

type opResult struct {
	op string
	d  time.Duration
}

func main() {
	fmt.Println("=== Benchmark: HTTP vs gRPC ===")

	cfg := loadConfig()
	fmt.Printf(
		"Config -> iterations: %d, concurrency: %d, warmup: %d, rpc-timeout: %s, http: %s, grpc: %s\n",
		cfg.Iterations, cfg.Concurrency, cfg.Warmup, cfg.RPCTimeout, cfg.HTTPBaseURL, cfg.GRPCAddress,
	)

	httpStats, err := measureHTTPBatch(cfg)
	if err != nil {
		log.Fatalf("HTTP benchmark failed: %v", err)
	}

	grpcStats, err := measureGRPCBatch(cfg)
	if err != nil {
		log.Fatalf("gRPC benchmark failed: %v", err)
	}

	fmt.Println()
	fmt.Println("HTTP results:")
	printStats(httpStats)

	fmt.Println()
	fmt.Println("gRPC results:")
	printStats(grpcStats)
}

func printStats(results map[string]stats) {
	for _, op := range operationsOrder {
		stat, ok := results[op]
		if !ok || stat.Avg == 0 {
			continue
		}
		fmt.Printf("  %-6s avg=%v | min=%v | max=%v\n", op, stat.Avg, stat.Min, stat.Max)
	}
}

func loadConfig() benchConfig {
	return benchConfig{
		HTTPBaseURL: strings.TrimRight(getEnv("BENCH_HTTP_BASE_URL", defaultHTTPBaseURL), "/"),
		GRPCAddress: getEnv("BENCH_GRPC_ADDR", defaultGRPCAddress),
		Iterations:  getEnvInt("BENCH_ITERATIONS", defaultIterations),
		Concurrency: getEnvInt("BENCH_CONCURRENCY", defaultConcurrency),
		Warmup:      getEnvInt("BENCH_WARMUP", defaultWarmup),
		RPCTimeout:  time.Duration(getEnvInt("BENCH_RPC_TIMEOUT_MS", defaultRPCTimeoutMs)) * time.Millisecond,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

// -------------------- HTTP --------------------

func measureHTTPBatch(cfg benchConfig) (map[string]stats, error) {
	transport := &http.Transport{
		MaxIdleConns:        1024,
		MaxIdleConnsPerHost: 1024,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.RPCTimeout,
	}
	usersURL := cfg.HTTPBaseURL + "/users"

	// Warm-up: few create and delete operations without measurements
	for i := 0; i < cfg.Warmup; i++ {
		payload := makeUserPayload("warm-http", httpEmailDomain, i, createDataSalt)
		u, _ := httpCreateUser(client, usersURL, payload)
		_ = httpDeleteUser(client, usersURL, u.ID)
	}

	collector := newCollector(operationsOrder...)
	out := make(chan opResult, cfg.Iterations*len(operationsOrder))
	var wg sync.WaitGroup

	per := (cfg.Iterations + cfg.Concurrency - 1) / cfg.Concurrency
	for w := 0; w < cfg.Concurrency; w++ {
		start := w * per
		end := min((w+1)*per, cfg.Iterations)
		if start >= end {
			break
		}
		wg.Add(1)
		go func(a, b int) {
			defer wg.Done()
			runHTTPWorker(client, usersURL, a, b, cfg.RPCTimeout, out)
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for r := range out {
		collector.add(r.op, r.d)
	}

	return collector.stats(), nil
}

func runHTTPWorker(client *http.Client, usersURL string, idxStart, idxEnd int, to time.Duration, out chan<- opResult) {
	for i := idxStart; i < idxEnd; i++ {
		createPayload := makeUserPayload("http-user", httpEmailDomain, i, createDataSalt)

		// create
		t0 := time.Now()
		created, err := httpCreateUser(client, usersURL, createPayload)
		if err != nil {
			continue
		}
		out <- opResult{"create", time.Since(t0)}

		// update
		updatePayload := makeUserPayload("http-user", httpEmailDomain, i, updateDataSalt)
		t0 = time.Now()
		if _, err := httpUpdateUser(client, usersURL, created.ID, updatePayload); err == nil {
			out <- opResult{"update", time.Since(t0)}
		}

		// get
		t0 = time.Now()
		if _, err := httpGetUser(client, usersURL, created.ID); err == nil {
			out <- opResult{"get", time.Since(t0)}
		}

		// delete
		t0 = time.Now()
		if err := httpDeleteUser(client, usersURL, created.ID); err == nil {
			out <- opResult{"delete", time.Since(t0)}
		}
	}
}

func httpCreateUser(client *http.Client, usersURL string, payload wireUser) (wireUser, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return wireUser{}, err
	}

	resp, err := client.Post(usersURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return wireUser{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return wireUser{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var created wireUser
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return wireUser{}, err
	}
	return created, nil
}

func httpUpdateUser(client *http.Client, usersURL, id string, payload wireUser) (wireUser, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return wireUser{}, err
	}

	req, err := http.NewRequest(http.MethodPut, usersURL+"/"+id, bytes.NewReader(body))
	if err != nil {
		return wireUser{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return wireUser{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return wireUser{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var updated wireUser
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return wireUser{}, err
	}
	return updated, nil
}

func httpGetUser(client *http.Client, usersURL, id string) (wireUser, error) {
	req, err := http.NewRequest(http.MethodGet, usersURL+"/"+id, nil)
	if err != nil {
		return wireUser{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return wireUser{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return wireUser{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var user wireUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return wireUser{}, err
	}
	return user, nil
}

func httpDeleteUser(client *http.Client, usersURL, id string) error {
	req, err := http.NewRequest(http.MethodDelete, usersURL+"/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// -------------------- gRPC --------------------

func measureGRPCBatch(cfg benchConfig) (map[string]stats, error) {
	conn, err := grpc.Dial(
		cfg.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)

	// Warm-up
	for i := 0; i < cfg.Warmup; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RPCTimeout)
		payload := makeUserPayload("warm-grpc", grpcEmailDomain, i, createDataSalt)
		u, _ := client.CreateUser(ctx, payload.toCreateRequest())
		cancel()
		if u != nil {
			ctx, c2 := context.WithTimeout(context.Background(), cfg.RPCTimeout)
			_, _ = client.DeleteUser(ctx, &userpb.DeleteUserRequest{Id: u.GetUser().GetId()})
			c2()
		}
	}

	collector := newCollector(operationsOrder...)
	out := make(chan opResult, cfg.Iterations*len(operationsOrder))
	var wg sync.WaitGroup

	per := (cfg.Iterations + cfg.Concurrency - 1) / cfg.Concurrency
	for w := 0; w < cfg.Concurrency; w++ {
		start := w * per
		end := min((w+1)*per, cfg.Iterations)
		if start >= end {
			break
		}
		wg.Add(1)
		go func(a, b int) {
			defer wg.Done()
			runGRPCWorker(client, a, b, cfg.RPCTimeout, out)
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for r := range out {
		collector.add(r.op, r.d)
	}

	return collector.stats(), nil
}

func runGRPCWorker(client userpb.UserServiceClient, idxStart, idxEnd int, to time.Duration, out chan<- opResult) {
	for i := idxStart; i < idxEnd; i++ {
		createPayload := makeUserPayload("grpc-user", grpcEmailDomain, i, createDataSalt)

		// create
		t0 := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), to)
		created, err := client.CreateUser(ctx, createPayload.toCreateRequest())
		cancel()
		if err != nil {
			continue
		}
		out <- opResult{"create", time.Since(t0)}

		id := created.GetUser().GetId()
		updatePayload := makeUserPayload("grpc-user", grpcEmailDomain, i, updateDataSalt)

		// update
		t0 = time.Now()
		ctx, cancel = context.WithTimeout(context.Background(), to)
		_, err = client.UpdateUser(ctx, updatePayload.toUpdateRequest(id))
		cancel()
		if err == nil {
			out <- opResult{"update", time.Since(t0)}
		}

		// get
		t0 = time.Now()
		ctx, cancel = context.WithTimeout(context.Background(), to)
		_, err = client.GetUser(ctx, &userpb.GetUserRequest{Id: id})
		cancel()
		if err == nil {
			out <- opResult{"get", time.Since(t0)}
		}

		// delete
		t0 = time.Now()
		ctx, cancel = context.WithTimeout(context.Background(), to)
		_, err = client.DeleteUser(ctx, &userpb.DeleteUserRequest{Id: id})
		cancel()
		if err == nil {
			out <- opResult{"delete", time.Since(t0)}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
