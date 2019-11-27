package schedule

import (
	"github.com/go-playground/assert/v2"
	"log"
	"sync"
	"testing"
	"time"
)

func TestSchedule(t *testing.T) {

}

func TestSchedule_Delay(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0

	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(1 * time.Millisecond)
		i = i + 1
	}

	for i := 0; i < 100; i++ {
		sche.Delay(time.Duration(1000+i*10) * time.Millisecond).Do(f)
	}
	time.Sleep(3 * time.Second)
	{
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, 100, i)
	}
}

func TestSchedule_Every(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0

	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(1 * time.Millisecond)
		i = i + 1
	}
	sche.Every(100 * time.Millisecond).Do(f)
	time.Sleep(4550 * time.Millisecond)
	{
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, 45, i)
	}
}

func TestDelayJob_CancelDoing(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(50 * time.Millisecond)
		i = i + 1
	}
	job := sche.Delay(100 * time.Millisecond).Do(f)
	time.Sleep(120 * time.Millisecond)
	err := sche.Cancel(job)
	assert.NotEqual(t, err, nil)
}

func TestEveryJob_CancelDoing(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(5 * time.Millisecond)
		i = i + 1
	}
	job := sche.Every(50 * time.Millisecond).Do(f)
	time.Sleep(102 * time.Millisecond)
	err := sche.Cancel(job)
	if err != nil {
		log.Fatalln(err)
	}
}

func TestDelayJob_Cancel(t *testing.T) {
	delays := []string{}
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	temp := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(5 * time.Millisecond)
		i = i + 1
	}
	for i := 0; i < 100; i++ {
		jobid := sche.Delay(time.Duration(1000+i*10) * time.Millisecond).Do(f)
		delays = append(delays, jobid)
	}
	time.Sleep(2000 * time.Millisecond)
	for _, delay := range delays {
		err := sche.Cancel(delay)
		if err != nil {
			temp++
		}

	}
	time.Sleep(1 * time.Second)
	{
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, temp, i)
	}
}

func TestEveryJob_Cancel(t *testing.T) {
	everys := []string{}
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(1 * time.Millisecond)
		i = i + 1
	}
	for i := 0; i < 10; i++ {
		jobid := sche.Every(100 * time.Millisecond).Do(f)
		everys = append(everys, jobid)
	}
	time.Sleep(1510 * time.Millisecond)
	for _, every := range everys {
		err := sche.Cancel(every)
		if err != nil {
			t.Log(err)
		}
	}
	time.Sleep(1 * time.Second)
	{
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, i, 150)
	}
}

func TestJob_GetJobStats(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(2 * time.Millisecond)
		i = i + 1
	}
	jobs := []string{}
	for i := 0; i < 10; i++ {
		job1 := sche.Every(3 * time.Millisecond).Do(f)
		jobs = append(jobs, job1)
		job2 := sche.Delay(1 * time.Millisecond * time.Duration(i)).Do(f)
		jobs = append(jobs, job2)
	}
	time.Sleep(1 * time.Second)
	temp := 0
	for _, job := range jobs {
		sche.Cancel(job)
	}
	time.Sleep(30 * time.Millisecond)
	for _, job := range jobs {
		result, err := sche.Query(job)
		if err != nil {
			log.Fatal(err)
		} else {
			temp += result.FinishedTime
		}
	}
	{
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, temp, i)
	}
}

func TestJobReDo(t *testing.T) {
	sche := NewSchedule()
	lock := sync.Mutex{}
	i := 0
	f := func() {
		lock.Lock()
		defer lock.Unlock()
		time.Sleep(2 * time.Millisecond)
		log.Fatal("error")
		i = i + 1
	}
	f2 := func() {
	}
	task := sche.Every(300 * time.Millisecond)
	task.Do(f)
	task.Do(f2)
	time.Sleep(1 * time.Second)
	{
		lock.Lock()
		defer lock.Unlock()
		assert.NotEqual(t, 3, i)
	}
}
