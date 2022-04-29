package timedmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSectionFlush(t *testing.T) {
	tm := New[int, int](dCleanupTick)

	for i := 0; i < 5; i++ {
		tm.set(i, 0, 1, time.Hour)
	}
	for i := 0; i < 10; i++ {
		tm.set(i, 1, 1, time.Hour)
	}
	for i := 0; i < 12; i++ {
		tm.set(i, 2, 1, time.Hour)
	}
	tm.Section(2).Flush()
	assert.EqualValues(t, 15, len(tm.container))

	tm.Section(1).Flush()
	assert.EqualValues(t, 5, len(tm.container))

	tm.Section(0).Flush()
	assert.EqualValues(t, 0, len(tm.container))
}

func TestSectionIdent(t *testing.T) {
	tm := New[int, int](dCleanupTick)

	assert.EqualValues(t, 1, tm.Section(1).Ident())
	assert.EqualValues(t, 2, tm.Section(2).Ident())
	assert.EqualValues(t, 3, tm.Section(3).Ident())
}

func TestSectionSet(t *testing.T) {
	const key = "tKeySet"
	const val = "tValSet"
	const sec = 1

	tm := New[string, string](dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 20*time.Millisecond)
	assert.Equal(t, val, tm.get(key, sec).value)

	time.Sleep(40 * time.Millisecond)
	assert.Nil(t, tm.get(key, sec))
}

func TestSectionGetValue(t *testing.T) {
	const key = "tKeyGetVal"
	const val = "tValGetVal"
	const sec = 1

	tm := New[string, string](dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 50*time.Millisecond)

	assert.Equal(t, "", s.GetValue("keyNotExists"))

	assert.Equal(t, val, s.GetValue(key))

	time.Sleep(60 * time.Millisecond)

	assert.Equal(t, "", s.GetValue(key))

	s.Set(key, val, 1*time.Microsecond)
	time.Sleep(2 * time.Millisecond)
	assert.Equal(t, "", s.GetValue(key))
}

func TestSectionGetExpire(t *testing.T) {
	const key = "tKeyGetExp"
	const val = "tValGetExp"
	const sec = 1

	tm := New[string, string](dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, val, 50*time.Millisecond)
	ct := time.Now().Add(50 * time.Millisecond)

	_, err := s.GetExpires("keyNotExists")
	assert.ErrorIs(t, err, ErrKeyNotFound)

	exp, err := s.GetExpires(key)
	assert.Nil(t, err)
	assert.Less(t, ct.Sub(exp), 1*time.Millisecond)

	tm.Flush()
}

func TestSectionSetExpires(t *testing.T) {
	const key = "tKeyRef"
	const sec = 1

	tm := New[string, int](dCleanupTick)

	s := tm.Section(sec)

	if err := tm.SetExpires("notExistentKey", 1*time.Second); err != ErrKeyNotFound {
		t.Errorf("returned error should have been '%s', but was '%s'",
			ErrKeyNotFound.Error(), err.Error())
	}

	s.Set(key, 1, 12*time.Millisecond)
	assert.Nil(t, s.SetExpires(key, 50*time.Millisecond))

	time.Sleep(30 * time.Millisecond)
	assert.NotNil(t, tm.get(key, sec))

	time.Sleep(51 * time.Millisecond)
	assert.Nil(t, tm.get(key, sec))
}

func TestSectionContains(t *testing.T) {
	const key = "tKeyCont"
	const sec = 1

	tm := New[string, int](dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, 1, 30*time.Millisecond)

	assert.False(t, s.Contains("keyNotExists"))
	assert.True(t, s.Contains(key))

	time.Sleep(50 * time.Millisecond)
	assert.False(t, s.Contains(key))
}

func TestSectionRemove(t *testing.T) {
	const key = "tKeyRem"
	const sec = 1

	tm := New[string, int](dCleanupTick)

	s := tm.Section(sec)

	s.Set(key, 1, time.Hour)
	s.Remove(key)
	assert.Nil(t, tm.get(key, sec))
}

func TestSectionRefresh(t *testing.T) {
	const key = "tKeyRef"
	const sec = 1

	tm := New[string, int](dCleanupTick)

	s := tm.Section(sec)

	assert.ErrorIs(t, s.Refresh("keyNotExists", time.Hour), ErrKeyNotFound)

	s.Set(key, 1, 12*time.Millisecond)
	assert.Nil(t, s.Refresh(key, 50*time.Millisecond))

	time.Sleep(30 * time.Millisecond)
	assert.NotNil(t, tm.get(key, sec))

	time.Sleep(100 * time.Millisecond)
	assert.Nil(t, tm.get(key, sec))
}

func TestSectionSize(t *testing.T) {
	tm := New[int, int](dCleanupTick)

	for i := 0; i < 20; i++ {
		tm.set(i, 0, 1, 50*time.Millisecond)
	}
	for i := 0; i < 25; i++ {
		tm.set(i, 1, 1, 50*time.Millisecond)
	}
	assert.EqualValues(t, 25, tm.Section(1).Size())
}

func TestSectionCallback(t *testing.T) {
	cb := new(CB[int])
	cb.On("Cb").Return()

	tm := New[int, int](dCleanupTick)

	tm.Section(1).Set(1, 3, 25*time.Millisecond, cb.Cb)

	time.Sleep(50 * time.Millisecond)
	assert.Nil(t, tm.get(1, 0))
	cb.AssertCalled(t, "Cb")
	assert.EqualValues(t, 3, cb.TestData().Get("v").Int())
}

func TestSectionSnapshot(t *testing.T) {
	tm := New[int, int](1 * time.Minute)

	for i := 0; i < 10; i++ {
		tm.set(i, i%2, i, 1*time.Minute)
	}

	m := tm.Section(1).Snapshot()

	assert.Len(t, m, 5)
	for i := 0; i < 10; i++ {
		if i%2 == 1 {
			assert.EqualValues(t, i, m[i])
		} else {
			assert.Equal(t, 0, m[i])
		}
	}
}
