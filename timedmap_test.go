package timedmap

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	dCleanupTick = 10 * time.Millisecond
)

func TestNew(t *testing.T) {
	tm := New(dCleanupTick)

	assert.NotNil(t, tm)
	assert.EqualValues(t, 0, len(tm.container))
}

func TestFlush(t *testing.T) {
	tm := New(dCleanupTick)

	for i := 0; i < 10; i++ {
		tm.set(i, 0, 1, time.Hour)
	}
	assert.EqualValues(t, 10, len(tm.container))
	tm.Flush()
	assert.EqualValues(t, 0, len(tm.container))
}

func TestSet(t *testing.T) {
	const key = "tKeySet"
	const val = "tValSet"

	tm := New(dCleanupTick)

	tm.Set(key, val, 20*time.Millisecond)
	if v := tm.get(key, 0); v == nil {
		t.Fatal("key was not set")
	} else if v.value.(string) != val {
		t.Fatal("value was not like set")
	}
	assert.Equal(t, val, tm.get(key, 0).value)

	time.Sleep(40 * time.Millisecond)
	assert.Nil(t, tm.get(key, 0))
}

func TestGetValue(t *testing.T) {
	const key = "tKeyGetVal"
	const val = "tValGetVal"

	tm := New(dCleanupTick)

	tm.Set(key, val, 50*time.Millisecond)
	assert.Nil(t, tm.GetValue("keyNotExists"))

	assert.Equal(t, val, tm.GetValue(key))

	time.Sleep(60 * time.Millisecond)

	assert.Nil(t, tm.GetValue(key))

	tm.Set(key, val, 1*time.Microsecond)
	time.Sleep(2 * time.Millisecond)
	assert.Nil(t, tm.GetValue(key))
}

func TestGetExpire(t *testing.T) {
	const key = "tKeyGetExp"
	const val = "tValGetExp"

	tm := New(dCleanupTick)

	tm.Set(key, val, 50*time.Millisecond)
	ct := time.Now().Add(50 * time.Millisecond)

	_, err := tm.GetExpires("keyNotExists")
	assert.ErrorIs(t, err, ErrKeyNotFound)

	exp, err := tm.GetExpires(key)
	assert.Nil(t, err)
	assert.Less(t, ct.Sub(exp), 1*time.Millisecond)
}

func TestSetExpire(t *testing.T) {
	const key = "tKeyRef"

	tm := New(dCleanupTick)

	err := tm.Refresh("keyNotExists", time.Hour)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	err = tm.SetExpire("notExistentKey", 1*time.Second)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	tm.Set(key, 1, 12*time.Millisecond)
	err = tm.SetExpire(key, 50*time.Millisecond)
	assert.Nil(t, err)

	time.Sleep(30 * time.Millisecond)
	assert.NotNil(t, tm.get(key, 0))

	time.Sleep(52 * time.Millisecond)
	assert.Nil(t, tm.get(key, 0))
}

func TestContains(t *testing.T) {
	const key = "tKeyCont"

	tm := New(dCleanupTick)

	tm.Set(key, 1, 30*time.Millisecond)

	assert.False(t, tm.Contains("keyNotExists"))
	assert.True(t, tm.Contains(key))

	time.Sleep(50 * time.Millisecond)
	assert.False(t, tm.Contains(key))
}

func TestRemove(t *testing.T) {
	const key = "tKeyRem"

	tm := New(dCleanupTick)

	tm.Set(key, 1, time.Hour)
	tm.Remove(key)

	assert.Nil(t, tm.get(key, 0))
}

func TestRefresh(t *testing.T) {
	const key = "tKeyRef"

	tm := New(dCleanupTick)

	err := tm.Refresh("keyNotExists", time.Hour)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	tm.Set(key, 1, 12*time.Millisecond)
	assert.Nil(t, tm.Refresh(key, 50*time.Millisecond))

	time.Sleep(30 * time.Millisecond)
	assert.NotNil(t, tm.get(key, 0))

	time.Sleep(100 * time.Millisecond)
	assert.Nil(t, tm.get(key, 0))
}

func TestSize(t *testing.T) {
	tm := New(dCleanupTick)

	for i := 0; i < 25; i++ {
		tm.Set(i, 1, 50*time.Millisecond)
	}
	assert.EqualValues(t, 25, tm.Size())
}

func TestCallback(t *testing.T) {
	cb := new(CB)
	cb.On("Cb").Return()

	tm := New(dCleanupTick)

	tm.Set(1, 3, 25*time.Millisecond, cb.Cb)

	time.Sleep(50 * time.Millisecond)
	assert.Nil(t, tm.get(1, 0))
	cb.AssertCalled(t, "Cb")
	assert.EqualValues(t, 3, cb.TestData().Get("v").Int())
}

func TestStopCleaner(t *testing.T) {
	tm := New(dCleanupTick)

	tm.StopCleaner()
	time.Sleep(10 * time.Millisecond)
}

func TestSnapshot(t *testing.T) {
	tm := New(1 * time.Minute)

	for i := 0; i < 10; i++ {
		tm.set(i, 0, i, 1*time.Minute)
	}

	m := tm.Snapshot()

	assert.Len(t, m, 10)
	for i := 0; i < 10; i++ {
		assert.EqualValues(t, i, m[i])
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	tm := New(dCleanupTick)

	go func() {
		for {
			for i := 0; i < 100; i++ {
				tm.Set(i, i, 2*time.Second)
			}
		}
	}()

	// Wait 10 mills before read cycle starts so that
	// it does not start before the first values are
	// set to the map.
	time.Sleep(10 * time.Millisecond)
	go func() {
		for {
			for i := 0; i < 100; i++ {
				v := tm.GetValue(i)
				assert.EqualValues(t, i, v)
			}
		}
	}()

	time.Sleep(1 * time.Second)
}

func TestGetExpiredConcurrent(t *testing.T) {
	tm := New(dCleanupTick)

	wg := sync.WaitGroup{}
	for i := 0; i < 50000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.Set(1, 1, 0)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.GetValue(1)
		}()
	}

	wg.Wait()
}

func TestExternalTicker(t *testing.T) {
	const key = "tKeySet"
	const val = "tValSet"

	ticker := time.NewTicker(dCleanupTick)
	tm := New(0, ticker.C)

	tm.Set(key, val, 20*time.Millisecond)
	assert.Equal(t, val, tm.get(key, 0).value)

	time.Sleep(40 * time.Millisecond)
	assert.Nil(t, tm.get(key, 0))
}

func TestBeforeCleanup(t *testing.T) {
	const key, value = 1, 2

	tm := New(1 * time.Hour)

	tm.Set(key, value, 5*time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	_, ok := tm.GetValue(key).(int)
	assert.False(t, ok)
}

// ----------------------------------------------------------
// --- BENCHMARKS ---

func BenchmarkSetValues(b *testing.B) {
	tm := New(1 * time.Minute)
	for n := 0; n < b.N; n++ {
		tm.Set(n, n, 1*time.Hour)
	}
}

func BenchmarkSetGetValues(b *testing.B) {
	tm := New(1 * time.Minute)
	for n := 0; n < b.N; n++ {
		tm.Set(n, n, 1*time.Hour)
		tm.GetValue(n)
	}
}

func BenchmarkSetGetRemoveValues(b *testing.B) {
	tm := New(1 * time.Minute)
	for n := 0; n < b.N; n++ {
		tm.Set(n, n, 1*time.Hour)
		tm.GetValue(n)
		tm.Remove(n)
	}
}

func BenchmarkSetGetSameKey(b *testing.B) {
	tm := New(1 * time.Minute)
	for n := 0; n < b.N; n++ {
		tm.Set(1, n, 1*time.Hour)
		tm.GetValue(1)
	}
}

// ----------------------------------------------------------
// --- UTILS ---

type CB struct {
	mock.Mock
}

func (cb *CB) Cb(v interface{}) {
	cb.TestData().Set("v", v)
	cb.Called()
}
