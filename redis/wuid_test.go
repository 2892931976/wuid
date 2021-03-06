package wuid

import (
	"flag"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edwingeng/wuid/internal"
	"github.com/go-redis/redis"
)

var bRedisCluster = flag.Bool("cluster", false, "")

type simpleLogger struct{}

func (this *simpleLogger) Info(args ...interface{}) {}
func (this *simpleLogger) Warn(args ...interface{}) {}

var sl = &simpleLogger{}

func getRedisConfig() (string, string, string) {
	return "127.0.0.1:6379", "", "wuid"
}

func getRedisClusterConfig() ([]string, string, string) {
	return []string{"127.0.0.1:6379", "127.0.0.1:6380", "127.0.0.1:6381"}, "", "wuid"
}

func TestWUID_LoadH24FromRedis(t *testing.T) {
	if *bRedisCluster {
		return
	}

	addr, pass, key := getRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
	})
	_, err := client.Del(key).Result()
	if err != nil {
		t.Fatal(err)
	}
	err = client.Close()
	if err != nil {
		t.Fatal(err)
	}

	g := NewWUID("default", sl)
	for i := 0; i < 1000; i++ {
		err = g.LoadH24FromRedis(getRedisConfig())
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH24FromRedis_Error(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", sl)
	addr, pass, key := getRedisConfig()

	if g.LoadH24FromRedis("", pass, key) == nil {
		t.Fatal("addr is not properly checked")
	}
	if g.LoadH24FromRedis(addr, pass, "") == nil {
		t.Fatal("key is not properly checked")
	}

	if err := g.LoadH24FromRedis("127.0.0.1:30000", pass, key); err == nil {
		t.Fatal("LoadH24FromRedis should fail when is address is invalid")
	}
}

func TestWUID_LoadH24FromRedisCluster(t *testing.T) {
	if !*bRedisCluster {
		return
	}

	addrs, pass, key := getRedisClusterConfig()
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: pass,
	})
	_, err := client.Del(key).Result()
	if err != nil {
		t.Fatal(err)
	}
	err = client.Close()
	if err != nil {
		t.Fatal(err)
	}

	g := NewWUID("default", sl)
	for i := 0; i < 1000; i++ {
		err = g.LoadH24FromRedisCluster(getRedisClusterConfig())
		if err != nil {
			t.Fatal(err)
		}
		v := (uint64(i) + 1) << 40
		if atomic.LoadUint64(&g.w.N) != v {
			t.Fatalf("g.w.N is %d, while it should be %d. i: %d", atomic.LoadUint64(&g.w.N), v, i)
		}
		for j := 0; j < rand.Intn(10); j++ {
			g.Next()
		}
	}
}

func TestWUID_LoadH24FromRedisCluster_Error(t *testing.T) {
	if !*bRedisCluster {
		return
	}

	g := NewWUID("default", sl)
	addrs, pass, key := getRedisClusterConfig()

	if g.LoadH24FromRedisCluster([]string{}, pass, key) == nil {
		t.Fatal("addr is not properly checked")
	}
	if g.LoadH24FromRedisCluster(addrs, pass, "") == nil {
		t.Fatal("key is not properly checked")
	}

	if err := g.LoadH24FromRedisCluster([]string{"127.0.0.1:30000"}, pass, key); err == nil {
		t.Fatal("LoadH24FromRedisCluster should fail when is address is invalid")
	}
}

func TestWUID_Next_Renew(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", nil)
	err := g.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}

	n1 := g.Next()
	kk := ((internal.CriticalValue + internal.RenewInterval) & ^internal.RenewInterval) - 1

	g.w.Reset((n1 >> 40 << 40) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n2 := g.Next()

	g.w.Reset((n2 >> 40 << 40) | kk)
	g.Next()
	time.Sleep(time.Millisecond * 200)
	n3 := g.Next()

	if n2>>40 == n1>>40 || n3>>40 == n2>>40 {
		t.Fatalf("the renew mechanism does not work as expected: %x, %x, %x", n1>>40, n2>>40, n3>>40)
	}
}

func TestWithSection(t *testing.T) {
	if *bRedisCluster {
		return
	}

	g := NewWUID("default", sl, WithSection(15))
	err := g.LoadH24FromRedis(getRedisConfig())
	if err != nil {
		t.Fatal(err)
	}
	if g.Next()>>60 != 15 {
		t.Fatal("WithSection does not work as expected")
	}
}

func Example() {
	// Setup
	g := NewWUID("default", nil)
	_ = g.LoadH24FromRedis("127.0.0.1:6379", "", "wuid")

	// Generate
	for i := 0; i < 10; i++ {
		fmt.Printf("%#016x\n", g.Next())
	}
}
