package main

import (
	"fmt"
	"github.com/FLYyyyy2016/schedule"
	"time"
)

func main() {
	sche := schedule.NewSchedule()
	c := sche.Every(1 * time.Second).Do(func() {
		fmt.Println("hello 1s every,print 2 times")
	})
	b := sche.Delay(2 * time.Second).Do(func() {
		fmt.Println("hello 2s after")
	})
	a := sche.Delay(3 * time.Second).Do(func() {
		fmt.Println("this msg will not print")
	})
	time.Sleep(2500 * time.Millisecond)
	sche.Cancel(a)
	sche.Cancel(b)
	sche.Cancel(c)
	time.Sleep(1 * time.Second)

	status, _ := sche.Query(a)
	fmt.Println(status)
	status, _ = sche.Query(b)
	fmt.Println(status)
	status, _ = sche.Query(c)
	fmt.Println(status)

	time.Sleep(1 * time.Second)
}
