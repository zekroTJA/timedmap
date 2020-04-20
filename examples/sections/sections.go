package main

import (
	"fmt"
	"log"
	"time"

	"github.com/zekroTJA/timedmap"
)

type myData struct {
	data string
}

func main() {

	// Creates a new timed map which scans for
	// expired keys every 1 second
	tm := timedmap.New(1 * time.Second)

	// Get sections 0 and 1
	sec0 := tm.Section(0)
	sec1 := tm.Section(1)

	// set value for key 'hey' in section 0
	sec0.Set("hey", 213, 3*time.Second, func(v interface{}) {
		log.Println("key-value pair of 'hey' has expired")
	})

	// set value for key 'ho' in section 1
	sec1.Set("ho", &myData{data: "ho"}, 4*time.Second, func(v interface{}) {
		log.Println("key-value pair of 'ho' has expired")
	})

	// Print values
	printKeyVal(sec0, "hey")
	printKeyVal(sec0, "ho")
	printKeyVal(sec1, "hey")
	printKeyVal(sec1, "ho")

	fmt.Println("-----------------")
	fmt.Println("► wait for 5 secs")

	// Wait for 5 seconds
	// During this time the main thread is blocked, the
	// key-value pairs of "hey" and "ho" will be expired
	time.Sleep(5 * time.Second)

	fmt.Println("-----------------")

	// Print values after 5 seconds
	printKeyVal(sec0, "hey")
	printKeyVal(sec0, "ho")
	printKeyVal(sec1, "hey")
	printKeyVal(sec1, "ho")
}

func printKeyVal(s timedmap.Section, key interface{}) {
	d := s.GetValue(key)
	if d == nil {
		log.Printf(
			"data expired or section [%d] does not contain a value for '%v'",
			s.Ident(), key)
		return
	}

	log.Printf("[%d]%v = %v\n", s.Ident(), key, d)
}
