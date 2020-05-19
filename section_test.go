package timedmap

import (
	"testing"
	"time"
)

func TestSectionFlush(t *testing.T) {
	tm := New(dCleanupTick)

	for i := 0; i < 10; i++ {
		tm.set(i, 0, 1, time.Hour)
	}
	for i := 0; i < 10; i++ {
		tm.set(i, 1, 1, time.Hour)
	}
	for i := 0; i < 10; i++ {
		tm.set(i, 2, 1, time.Hour)
	}
	tm.Section(2).Flush()
	if s := len(tm.container); s > 20 {
		t.Fatalf("size was %d > 20", s)
	}
}

func TestSectionSet(t *testing.T) {
	const key = "tKeySet"
	const val = "tValSet"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 20*time.Millisecond)
	if v := tm.get(key, sec); v == nil {
		t.Fatal("key was not set")
	} else if v.value.(string) != val {
		t.Fatal("value was not like set")
	}
	time.Sleep(40 * time.Millisecond)
	if v := tm.get(key, sec); v != nil {
		t.Fatal("key was not deleted after expire")
	}

	tm.Flush()
}

func TestSectionGetValue(t *testing.T) {
	const key = "tKeyGetVal"
	const val = "tValGetVal"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 50*time.Millisecond)

	if s.GetValue("keyNotExists") != nil {
		t.Fatal("non existent key was not nil")
	}

	v := s.GetValue(key)
	if v == nil {
		t.Fatal("value was nil")
	}
	if vStr := v.(string); vStr != val {
		t.Fatalf("got value was %s != 'tValGetVal'", vStr)
	}

	time.Sleep(60 * time.Millisecond)

	v = s.GetValue(key)
	if v != nil {
		t.Fatal("key was not deleted after expiration time")
	}

	s.Set(key, val, 1*time.Microsecond)
	time.Sleep(2 * time.Millisecond)
	if s.GetValue(key) != nil {
		t.Fatal("expired key was not removed by get func")
	}

	tm.Flush()
}

func TestSectionGetExpire(t *testing.T) {
	const key = "tKeyGetExp"
	const val = "tValGetExp"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 50*time.Millisecond)
	ct := time.Now().Add(50 * time.Millisecond)

	if _, err := s.GetExpires("keyNotExists"); err.Error() != "key not found" {
		t.Fatal("err was not 'key not found': ", err)
	}

	exp, err := s.GetExpires(key)
	if err != nil {
		t.Fatal(err)
	}
	if d := ct.Sub(exp); d > 1*time.Millisecond {
		t.Fatalf("expire date diff was %d > 1 millisecond", d)
	}

	tm.Flush()
}

func TestSectionContains(t *testing.T) {
	const key = "tKeyCont"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, 1, 30*time.Millisecond)

	if s.Contains("keyNotExists") {
		t.Fatal("non existing key was detected as containing")
	}

	if !s.Contains(key) {
		t.Fatal("containing key was detected as not containing")
	}

	time.Sleep(50 * time.Millisecond)
	if s.Contains(key) {
		t.Fatal("expired key was detected as containing")
	}

	tm.Flush()
}

func TestSectionRemove(t *testing.T) {
	const key = "tKeyRem"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, 1, time.Hour)
	s.Remove(key)

	if v := tm.get(key, sec); v != nil {
		t.Fatal("key still exists after remove")
	}

	tm.Flush()
}

func TestSectionRefresh(t *testing.T) {
	const key = "tKeyRef"
	const sec = 1

	tm := New(dCleanupTick)

	s := tm.Section(sec)

	if err := s.Refresh("keyNotExists", time.Hour); err == nil || err.Error() != "key not found" {
		t.Fatalf("error on non existing key was %v != 'key not found'", err)
	}

	s.Set(key, 1, 12*time.Millisecond)
	if err := s.Refresh(key, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)
	if v := tm.get(key, sec); v == nil {
		t.Fatal("key was not refreshed")
	}

	time.Sleep(100 * time.Millisecond)
	if v := tm.get(key, sec); v != nil {
		t.Fatal("key was not deleted after refreshed time")
	}

	tm.Flush()
}

func TestSectionSize(t *testing.T) {
	tm := New(dCleanupTick)

	for i := 0; i < 20; i++ {
		tm.set(i, 0, 1, 50*time.Millisecond)
	}
	for i := 0; i < 25; i++ {
		tm.set(i, 1, 1, 50*time.Millisecond)
	}
	if s := tm.Section(1).Size(); s != 25 {
		t.Fatalf("size was %d != 25", s)
	}

	tm.Flush()
}

func TestSectionCallback(t *testing.T) {
	tm := New(dCleanupTick)

	var cbCalled bool
	tm.Section(1).Set(1, 3, 25*time.Millisecond, func(v interface{}) {
		cbCalled = true
	})

	time.Sleep(50 * time.Millisecond)
	if !cbCalled {
		t.Fatal("callback has not been called")
	}
	if v := tm.get(1, 0); v != nil {
		t.Fatal("key was not deleted after expire time")
	}
}
