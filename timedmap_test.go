package timedmap

import (
	"testing"
	"time"
)

const (
	dCleanupTick = 10 * time.Millisecond
)

var tm *TimedMap

func TestMain(m *testing.M) {
	tm = New(dCleanupTick)
	m.Run()
}

func TestNew(t *testing.T) {
	if tm == nil {
		t.Fatal("TimedMap was nil")
	}
	if s := len(tm.container); s != 0 {
		t.Fatalf("map size was %d != 0", s)
	}
}

func TestFlush(t *testing.T) {
	for i := 0; i < 10; i++ {
		tm.container[i] = &element{
			value:   1,
			expires: time.Now().Add(time.Hour),
		}
	}
	tm.Flush()
	if s := len(tm.container); s > 0 {
		t.Fatalf("size was %d > 0", s)
	}
}

func TestSet(t *testing.T) {
	key := "tKeySet"
	val := "tValSet"

	tm.Set(key, val, 20*time.Millisecond)
	if v, ok := tm.container[key]; !ok {
		t.Fatal("key was not set")
	} else if v.value.(string) != val {
		t.Fatal("value was not like set")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := tm.container[key]; ok {
		t.Fatal("key was not deleted after expire")
	}

	tm.Flush()
}

func TestGetValue(t *testing.T) {
	key := "tKeyGetVal"
	val := "tValGetVal"

	tm.Set(key, val, 50*time.Millisecond)

	if tm.GetValue("keyNotExists") != nil {
		t.Fatal("non existent key was not nil")
	}

	v := tm.GetValue(key)
	if v == nil {
		t.Fatal("value was nil")
	}
	if vStr := v.(string); vStr != val {
		t.Fatalf("got value was %s != 'tValGetVal'", vStr)
	}

	time.Sleep(60 * time.Millisecond)

	v = tm.GetValue(key)
	if v != nil {
		t.Fatal("key was not deleted after expiration time")
	}

	tm.Set(key, val, 1*time.Microsecond)
	time.Sleep(2 * time.Millisecond)
	if tm.GetValue(key) != nil {
		t.Fatal("expired key was not removed by get func")
	}

	tm.Flush()
}

func TestGetExpire(t *testing.T) {
	key := "tKeyGetExp"
	val := "tValGetExp"

	tm.Set(key, val, 50*time.Millisecond)
	ct := time.Now().Add(50 * time.Millisecond)

	if _, err := tm.GetExpires("keyNotExists"); err.Error() != "key not found" {
		t.Fatal("err was not 'key not found': ", err)
	}

	exp, err := tm.GetExpires(key)
	if err != nil {
		t.Fatal(err)
	}
	if d := ct.Sub(exp); d > 1*time.Millisecond {
		t.Fatalf("expire date diff was %d > 1 millisecond", d)
	}

	tm.Flush()
}

func TestContains(t *testing.T) {
	key := "tKeyCont"

	tm.Set(key, 1, 30*time.Millisecond)

	if tm.Contains("keyNotExists") {
		t.Fatal("non existing key was detected as containing")
	}

	if !tm.Contains(key) {
		t.Fatal("containing key was detected as not containing")
	}

	time.Sleep(50 * time.Millisecond)
	if tm.Contains(key) {
		t.Fatal("expired key was detected as containing")
	}

	tm.Flush()
}

func TestRemove(t *testing.T) {
	key := "tKeyRem"

	tm.Set(key, 1, time.Hour)
	tm.Remove(key)

	if _, ok := tm.container[key]; ok {
		t.Fatal("key still exists after remove")
	}

	tm.Flush()
}

func TestRefresh(t *testing.T) {
	key := "tKeyRef"

	if err := tm.Refresh("keyNotExists", time.Hour); err == nil || err.Error() != "key not found" {
		t.Fatalf("error on non existing key was %v != 'key not found'", err)
	}

	tm.Set(key, 1, 12*time.Millisecond)
	if err := tm.Refresh(key, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := tm.container[key]; !ok {
		t.Fatal("key was not refreshed")
	}

	time.Sleep(100 * time.Millisecond)
	if _, ok := tm.container[key]; ok {
		t.Fatal("key was not deleted after refreshed time")
	}

	tm.Flush()
}

func TestSize(t *testing.T) {
	for i := 0; i < 25; i++ {
		tm.Set(i, 1, 50*time.Millisecond)
	}
	if s := tm.Size(); s != 25 {
		t.Fatalf("size was %d != 25", s)
	}

	tm.Flush()
}

func TestCallback(t *testing.T) {
	var cbCalled bool
	tm.Set(1, 3, 25*time.Millisecond, func(v interface{}) {
		cbCalled = true
	})

	time.Sleep(50 * time.Millisecond)
	if !cbCalled {
		t.Fatal("callback has not been called")
	}
	if _, ok := tm.container[1]; ok {
		t.Fatal("key was not deleted after expire time")
	}
}

func TestStopCleaner(t *testing.T) {
	tm.StopCleaner()
}
