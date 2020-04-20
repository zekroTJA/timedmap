package timedmap

import (
	"errors"
	"time"
)

// Section defines a sectioned access
// wrapper of TimedMap.
type Section interface {
	Ident() int
	Set(key, value interface{}, expiresAfter time.Duration, cb ...callback)
	GetValue(key interface{}) interface{}
	GetExpires(key interface{}) (time.Time, error)
	Contains(key interface{}) bool
	Remove(key interface{})
	Refresh(key interface{}, d time.Duration) error
	Flush()
	Size() (i int)
}

// section wraps access to a specific
// section of the map.
type section struct {
	tm  *TimedMap
	sec int
}

// newSection creates a new Section instance
// wrapping the given TimedMap instance and
// section identifier.
func newSection(tm *TimedMap, sec int) *section {
	return &section{
		tm:  tm,
		sec: sec,
	}
}

// Ident returns the current sections identifier
func (s *section) Ident() int {
	return s.sec
}

// Set appends a key-value pair to the map or sets the value of
// a key. expiresAfter sets the expire time after the key-value pair
// will automatically be removed from the map.
func (s *section) Set(key, value interface{}, expiresAfter time.Duration, cb ...callback) {
	s.tm.set(key, s.sec, value, expiresAfter, cb...)
}

// GetValue returns an interface of the value of a key in the
// map. The returned value is nil if there is no value to the
// passed key or if the value was expired.
func (s *section) GetValue(key interface{}) interface{} {
	v := s.tm.get(key, s.sec)
	if v == nil {
		return nil
	}
	return v.value
}

// GetExpires returns the expire time of a key-value pair.
// If the key-value pair does not exist in the map or
// was expired, this will return an error object.
func (s *section) GetExpires(key interface{}) (time.Time, error) {
	v := s.tm.get(key, s.sec)
	if v == nil {
		return time.Time{}, errors.New("key not found")
	}
	return v.expires, nil
}

// Contains returns true, if the key exists in the map.
// false will be returned, if there is no value to the
// key or if the key-value pair was expired.
func (s *section) Contains(key interface{}) bool {
	return s.tm.get(key, s.sec) != nil
}

// Remove deletes a key-value pair in the map.
func (s *section) Remove(key interface{}) {
	s.tm.remove(key, s.sec)
}

// Refresh extends the expire time for a key-value pair
// about the passed duration. If there is no value to
// the key passed, this will return an error object.
func (s *section) Refresh(key interface{}, d time.Duration) error {
	return s.tm.refresh(key, s.sec, d)
}

// Flush deletes all key-value pairs of the section
// in the map.
func (s *section) Flush() {
	for k := range s.tm.container {
		if k.sec == s.sec {
			s.tm.remove(k.key, k.sec)
		}
	}
}

// Size returns the current number of key-value pairs
// existent in the section of the map.
func (s *section) Size() (i int) {
	for k := range s.tm.container {
		if k.sec == s.sec {
			i++
		}
	}
	return
}
