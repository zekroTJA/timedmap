package timedmap

import (
	"time"
)

// Section defines a sectioned access
// wrapper of TimedMap.
type Section[TKey comparable, TVal any] interface {

	// Ident returns the current sections identifier
	Ident() int

	// Set appends a key-value pair to the map or sets the value of
	// a key. expiresAfter sets the expire time after the key-value pair
	// will automatically be removed from the map.
	Set(key TKey, value TVal, expiresAfter time.Duration, cb ...Callback[TVal])

	// GetValue returns an interface of the value of a key in the
	// map. The returned value is nil if there is no value to the
	// passed key or if the value was expired.
	GetValue(key TKey) TVal

	// GetExpires returns the expire time of a key-value pair.
	// If the key-value pair does not exist in the map or
	// was expired, this will return an error object.
	GetExpires(key TKey) (time.Time, error)

	// SetExpires sets the expire time for a key-value
	// pair to the passed duration. If there is no value
	// to the key passed , this will return an error.
	SetExpires(key TKey, d time.Duration) error

	// Contains returns true, if the key exists in the map.
	// false will be returned, if there is no value to the
	// key or if the key-value pair was expired.
	Contains(key TKey) bool

	// Remove deletes a key-value pair in the map.
	Remove(key TKey)

	// Refresh extends the expire time for a key-value pair
	// about the passed duration. If there is no value to
	// the key passed, this will return an error.
	Refresh(key TKey, d time.Duration) error

	// Flush deletes all key-value pairs of the section
	// in the map.
	Flush()

	// Size returns the current number of key-value pairs
	// existent in the section of the map.
	Size() (i int)

	// Snapshot returns a new map which represents the
	// current key-value state of the internal container.
	Snapshot() map[TKey]TVal
}

// section wraps access to a specific
// section of the map.
type section[TKey comparable, TVal any] struct {
	tm  *TimedMap[TKey, TVal]
	sec int
}

// newSection creates a new Section instance
// wrapping the given TimedMap instance and
// section identifier.
func newSection[TKey comparable, TVal any](tm *TimedMap[TKey, TVal], sec int) *section[TKey, TVal] {
	return &section[TKey, TVal]{
		tm:  tm,
		sec: sec,
	}
}

func (s *section[TKey, TVal]) Ident() int {
	return s.sec
}

func (s *section[TKey, TVal]) Set(
	key TKey,
	value TVal,
	expiresAfter time.Duration,
	cb ...Callback[TVal],
) {
	s.tm.set(key, s.sec, value, expiresAfter, cb...)
}

func (s *section[TKey, TVal]) GetValue(key TKey) (val TVal) {
	v := s.tm.get(key, s.sec)
	if v != nil {
		val = v.value
	}
	return
}

func (s *section[TKey, TVal]) GetExpires(key TKey) (time.Time, error) {
	v := s.tm.get(key, s.sec)
	if v == nil {
		return time.Time{}, ErrKeyNotFound
	}
	return v.expires, nil
}

func (s *section[TKey, TVal]) SetExpires(key TKey, d time.Duration) error {
	return s.tm.setExpires(key, s.sec, d)
}

func (s *section[TKey, TVal]) Contains(key TKey) bool {
	return s.tm.get(key, s.sec) != nil
}

func (s *section[TKey, TVal]) Remove(key TKey) {
	s.tm.remove(key, s.sec)
}

func (s *section[TKey, TVal]) Refresh(key TKey, d time.Duration) error {
	return s.tm.refresh(key, s.sec, d)
}

func (s *section[TKey, TVal]) Flush() {
	for k := range s.tm.container {
		if k.sec == s.sec {
			s.tm.remove(k.key, k.sec)
		}
	}
}

func (s *section[TKey, TVal]) Size() (i int) {
	for k := range s.tm.container {
		if k.sec == s.sec {
			i++
		}
	}
	return
}

func (s *section[TKey, TVal]) Snapshot() map[TKey]TVal {
	return s.tm.getSnapshot(s.sec)
}
