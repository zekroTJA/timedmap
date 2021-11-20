package timedmap

import (
	"sync"
	"time"
)

type callback func(value interface{})

// TimedMap contains a map with all key-value pairs,
// and a timer, which cleans the map in the set
// tick durations from expired keys.
type TimedMap[TKey comparable, TVal any] struct {
	mtx         sync.RWMutex
	container   map[keyWrap[TKey]]*element[TVal]
	elementPool *sync.Pool

	cleanupTickTime time.Duration
	cleanerTicker   *time.Ticker
	cleanerStopChan chan bool
	cleanerRunning  bool
}

type keyWrap[TKey comparable] struct {
	sec int
	key TKey
}

// element contains the actual value as interface type,
// the thime when the value expires and an array of
// callbacks, which will be executed when the element
// expires.
type element[TVal any] struct {
	value   TVal
	expires time.Time
	cbs     []callback
}

// New creates and returns a new instance of TimedMap.
// The passed cleanupTickTime will be passed to the
// cleanup ticker, which iterates through the map and
// deletes expired key-value pairs.
//
// Optionally, you can also pass a custom <-chan time.Time
// which controls the cleanup cycle if you want to use
// a single syncronyzed timer or if you want to have more
// control over the cleanup loop.
//
// When passing 0 as cleanupTickTime and no tickerChan,
// the cleanup loop will not be started. You can call
// StartCleanerInternal or StartCleanerExternal to
// manually start the cleanup loop. These both methods
// can also be used to re-define the specification of
// the cleanup loop when already running if you want to.
func New[TKey comparable, TVal any](cleanupTickTime time.Duration, tickerChan ...<-chan time.Time) *TimedMap[TKey, TVal] {
	tm := &TimedMap[TKey, TVal]{
		container:       make(map[keyWrap[TKey]]*element[TVal]),
		cleanerStopChan: make(chan bool),
		elementPool: &sync.Pool{
			New: func() interface{} {
				return new(element[TVal])
			},
		},
	}

	if len(tickerChan) > 0 {
		tm.StartCleanerExternal(tickerChan[0])
	} else if cleanupTickTime > 0 {
		tm.StartCleanerInternal(cleanupTickTime)
	}

	return tm
}

// Section returns a sectioned subset of
// the timed map with the given section
// identifier i.
func (tm *TimedMap[TKey, TVal]) Section(i int) Section[TKey, TVal] {
	if i == 0 {
		return tm
	}
	return newSection(tm, i)
}

// Ident returns the current sections ident.
// In the case of the root object TimedMap,
// this is always 0.
func (tm *TimedMap[TKey, TVal]) Ident() int {
	return 0
}

// Set appends a key-value pair to the map or sets the value of
// a key. expiresAfter sets the expire time after the key-value pair
// will automatically be removed from the map.
func (tm *TimedMap[TKey, TVal]) Set(key TKey, value TVal, expiresAfter time.Duration, cb ...callback) {
	tm.set(key, 0, value, expiresAfter, cb...)
}

// GetValue returns an interface of the value of a key in the
// map. The returned value is nil if there is no value to the
// passed key or if the value was expired.
func (tm *TimedMap[TKey, TVal]) GetValue(key TKey) (val TVal) {
	v := tm.get(key, 0)
	if v != nil {
		val = v.value
	}
	return
}

// GetExpires returns the expire time of a key-value pair.
// If the key-value pair does not exist in the map or
// was expired, this will return an error object.
func (tm *TimedMap[TKey, TVal]) GetExpires(key TKey) (time.Time, error) {
	v := tm.get(key, 0)
	if v == nil {
		return time.Time{}, ErrKeyNotFound
	}
	return v.expires, nil
}

// SetExpire is deprecated.
// Please use SetExpires instead.
func (tm *TimedMap[TKey, TVal]) SetExpire(key TKey, d time.Duration) error {
	return tm.SetExpires(key, d)
}

// SetExpires sets the expire time for a key-value
// pair to the passed duration. If there is no value
// to the key passed , this will return an error.
func (tm *TimedMap[TKey, TVal]) SetExpires(key TKey, d time.Duration) error {
	return tm.setExpires(key, 0, d)
}

// Contains returns true, if the key exists in the map.
// false will be returned, if there is no value to the
// key or if the key-value pair was expired.
func (tm *TimedMap[TKey, TVal]) Contains(key TKey) bool {
	return tm.get(key, 0) != nil
}

// Remove deletes a key-value pair in the map.
func (tm *TimedMap[TKey, TVal]) Remove(key TKey) {
	tm.remove(key, 0)
}

// Refresh extends the expire time for a key-value pair
// about the passed duration. If there is no value to
// the key passed, this will return an error object.
func (tm *TimedMap[TKey, TVal]) Refresh(key TKey, d time.Duration) error {
	return tm.refresh(key, 0, d)
}

// Flush deletes all key-value pairs of the map.
func (tm *TimedMap[TKey, TVal]) Flush() {
	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	for k, v := range tm.container {
		tm.elementPool.Put(v)
		delete(tm.container, k)
	}
}

// Size returns the current number of key-value pairs
// existent in the map.
func (tm *TimedMap[TKey, TVal]) Size() int {
	return len(tm.container)
}

// StartCleanerInternal starts the cleanup loop controlled
// by an internal ticker with the given interval.
//
// If the cleanup loop is already running, it will be
// stopped and restarted using the new specification.
func (tm *TimedMap[TKey, TVal]) StartCleanerInternal(interval time.Duration) {
	if tm.cleanerRunning {
		tm.StopCleaner()
	}
	tm.cleanerTicker = time.NewTicker(interval)
	go tm.cleanupLoop(tm.cleanerTicker.C)
}

// StartCleanerExternal starts the cleanup loop controlled
// by the given initiator channel. This is useful if you
// want to have more control over the cleanup loop or if
// you want to sync up multiple timedmaps.
//
// If the cleanup loop is already running, it will be
// stopped and restarted using the new specification.
func (tm *TimedMap[TKey, TVal]) StartCleanerExternal(initiator <-chan time.Time) {
	if tm.cleanerRunning {
		tm.StopCleaner()
	}
	go tm.cleanupLoop(initiator)
}

// StopCleaner stops the cleaner go routine and timer.
// This should always be called after exiting a scope
// where TimedMap is used that the data can be cleaned
// up correctly.
func (tm *TimedMap[TKey, TVal]) StopCleaner() {
	if !tm.cleanerRunning {
		return
	}
	tm.cleanerStopChan <- true
	if tm.cleanerTicker != nil {
		tm.cleanerTicker.Stop()
	}
}

// Snapshot returns a new map which represents the
// current key-value state of the internal container.
func (tm *TimedMap[TKey, TVal]) Snapshot() map[TKey]TVal {
	return tm.getSnapshot(0)
}

// cleanupLoop holds the loop executing the cleanup
// when initiated by tc.
func (tm *TimedMap[TKey, TVal]) cleanupLoop(tc <-chan time.Time) {
	tm.cleanerRunning = true
	defer func() {
		tm.cleanerRunning = false
	}()

	for {
		select {
		case <-tc:
			tm.cleanUp()
		case <-tm.cleanerStopChan:
			return
		}
	}
}

// expireElement removes the specified key-value element
// from the map and executes all defined callback functions
func (tm *TimedMap[TKey, TVal]) expireElement(key TKey, sec int, v *element[TVal]) {
	for _, cb := range v.cbs {
		cb(v.value)
	}

	k := keyWrap[TKey]{
		sec: sec,
		key: key,
	}

	tm.elementPool.Put(v)
	delete(tm.container, k)
}

// cleanUp iterates trhough the map and expires all key-value
// pairs which expire time after the current time
func (tm *TimedMap[TKey, TVal]) cleanUp() {
	now := time.Now()

	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	for k, v := range tm.container {
		if now.After(v.expires) {
			tm.expireElement(k.key, k.sec, v)
		}
	}
}

// set sets the value for a key and section with the
// given expiration parameters
func (tm *TimedMap[TKey, TVal]) set(key TKey, sec int, val TVal, expiresAfter time.Duration, cb ...callback) {
	// re-use element when existent on this key
	if v := tm.getRaw(key, sec); v != nil {
		v.value = val
		v.expires = time.Now().Add(expiresAfter)
		v.cbs = cb
		return
	}

	k := keyWrap[TKey]{
		sec: sec,
		key: key,
	}

	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	v := tm.elementPool.Get().(*element[TVal])
	v.value = val
	v.expires = time.Now().Add(expiresAfter)
	v.cbs = cb
	tm.container[k] = v
}

// get returns an element object by key and section
// if the value has not already expired
func (tm *TimedMap[TKey, TVal]) get(key TKey, sec int) *element[TVal] {
	v := tm.getRaw(key, sec)

	if v == nil {
		return nil
	}

	if time.Now().After(v.expires) {
		tm.mtx.Lock()
		defer tm.mtx.Unlock()
		tm.expireElement(key, sec, v)
		return nil
	}

	return v
}

// getRaw returns the raw element object by key,
// not depending on expiration time
func (tm *TimedMap[TKey, TVal]) getRaw(key TKey, sec int) *element[TVal] {
	k := keyWrap[TKey]{
		sec: sec,
		key: key,
	}

	tm.mtx.RLock()
	v, ok := tm.container[k]
	tm.mtx.RUnlock()

	if !ok {
		return nil
	}

	return v
}

// remove removes an element from the map by giveb
// key and section
func (tm *TimedMap[TKey, TVal]) remove(key TKey, sec int) {
	k := keyWrap[TKey]{
		sec: sec,
		key: key,
	}

	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	v, ok := tm.container[k]
	if !ok {
		return
	}

	tm.elementPool.Put(v)
	delete(tm.container, k)
}

// refresh extends the lifetime of the given key in the
// given section by the duration d.
func (tm *TimedMap[TKey, TVal]) refresh(key TKey, sec int, d time.Duration) error {
	v := tm.get(key, sec)
	if v == nil {
		return ErrKeyNotFound
	}
	v.expires = v.expires.Add(d)
	return nil
}

// setExpires sets the lifetime of the given key in the
// given section to the duration d.
func (tm *TimedMap[TKey, TVal]) setExpires(key TKey, sec int, d time.Duration) error {
	v := tm.get(key, sec)
	if v == nil {
		return ErrKeyNotFound
	}
	v.expires = time.Now().Add(d)
	return nil
}

func (tm *TimedMap[TKey, TVal]) getSnapshot(sec int) (m map[TKey]TVal) {
	m = make(map[TKey]TVal)

	tm.mtx.RLock()
	defer tm.mtx.RUnlock()

	for k, v := range tm.container {
		if k.sec == sec {
			m[k.key] = v.value
		}
	}

	return
}
