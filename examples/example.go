package main

import (
	"log"
	"time"

	"github.com/zekroTJA/timedmap"
)

func main() {

	tm := timedmap.New(1 * time.Second)

	tm.Set("hey", 213, 3*time.Second)

	printKeyVal(tm, "hey")

	time.Sleep(5 * time.Second)

	printKeyVal(tm, "hey")
}

func printKeyVal(tm *timedmap.TimedMap, key interface{}) {
	d := tm.GetValue(key)
	if d == nil {
		log.Println("data expired")
		return
	}

	dInt := d.(int)
	log.Printf("%v = %d\n", key, dInt)
}
