package schedule

import (
	"crypto/md5"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	jobPrepare jobStatus = iota
	jobRunning
	jobFinish
	jobCreating
	jobCancel
	jobNotExist
)

type Schedule struct {
	tasks map[string]Task
	sync.Mutex
}

type JobStats struct {
	jobStatus    jobStatus
	finishedTime int
}

type jobStatus uint

type jobIsRunningError struct{}

func (error jobIsRunningError) Error() string {
	return "job is running ,you can try it again"
}

type jobIsCreatingError struct{}

func (error jobIsCreatingError) Error() string {
	return "job is Creating ,you can try it again"
}

type jobIsCancelError struct{}

func (error jobIsCancelError) Error() string {
	return "job is canceled "
}

type jobIsFinishError struct{}

func (error jobIsFinishError) Error() string {
	return "job is finished"
}

type jobNotExistError struct{}

func (error jobNotExistError) Error() string {
	return "job not exist ,you can try it again"
}

func NewSchedule() *Schedule {
	return &Schedule{tasks: make(map[string]Task)}
}

func (sche *Schedule) Delay(duration time.Duration) Task {
	sche.Lock()
	defer sche.Unlock()
	newJob := DelayJob{
		Job: Job{JobId: nextId(), close: make(chan struct{}), duration: duration},
	}
	newJob.setStatus(jobCreating)
	sche.tasks[newJob.JobId] = &newJob
	return &newJob
}

func (sche *Schedule) Every(duration time.Duration) Task {
	sche.Lock()
	defer sche.Unlock()
	newJob := EveryJob{
		Job: Job{JobId: nextId(), close: make(chan struct{}), duration: duration},
	}
	newJob.setStatus(jobCreating)
	sche.tasks[newJob.JobId] = &newJob
	return &newJob
}

func (sche *Schedule) Cancel(jobId string) error {
	sche.Lock()
	if task, ok := sche.tasks[jobId]; ok {
		sche.Unlock()
		return task.Cancel()
	} else {
		sche.Unlock()
		return jobNotExistError{}
	}
}

type Task interface {
	Do(func()) string
	GetId() string
	Cancel() error
}

type DelayJob struct {
	Job

	finish chan struct{}
}

type EveryJob struct {
	Job
	finish chan int
}

type Job struct {
	sync.Mutex
	JobId    string
	status   jobStatus
	work     func()
	close    chan struct{}
	jobStats JobStats
	duration time.Duration
}

func (job *Job) setStatus(status jobStatus) {
	job.Lock()
	defer job.Unlock()
	job.status = status
	job.jobStats.jobStatus = status
}
func (job *Job) getStatus() jobStatus {
	job.Lock()
	defer job.Unlock()
	return job.status
}

func (job *DelayJob) Cancel() error {
	status := job.getStatus()
	switch status {
	case jobPrepare:
		job.close <- struct{}{}
	case jobRunning:
		return jobIsRunningError{}
	case jobCancel:
		return jobIsCancelError{}
	case jobCreating:
		return jobIsCreatingError{}
	case jobFinish:
		return jobIsFinishError{}
	}
	return nil
}

func (job *EveryJob) Cancel() error {
	status := job.getStatus()
	switch status {
	case jobPrepare:
		job.close <- struct{}{}
	case jobRunning:
		job.close <- struct{}{}
	case jobCancel:
		return jobIsCancelError{}
	case jobCreating:
		return jobIsCreatingError{}
	case jobFinish:
		return jobIsFinishError{}
	}
	return nil
}

func (job *Job) GetId() string {
	return job.JobId
}

func (job *DelayJob) Do(f func()) string {
	timer := time.NewTimer(job.duration)
	go func() {
		job.setStatus(jobPrepare)
		select {
		case <-timer.C:
			if job.getStatus() == jobPrepare {
				go func() {
					job.setStatus(jobRunning)
					f()
					job.finishOneTime()
					if job.getStatus() == jobRunning {
						job.setStatus(jobFinish)
					}
				}()
			}
		case <-job.close:
			timer.Stop()
			job.setStatus(jobCancel)
			return
		}
	}()
	return job.JobId
}

func (job *Job) finishOneTime() {
	job.Lock()
	defer job.Unlock()
	job.jobStats.finishedTime++
}

func (job *EveryJob) Do(f func()) string {
	timer := time.NewTicker(job.duration)
	go func() {
		job.setStatus(jobPrepare)
		for {
			select {
			case <-timer.C:
				if job.getStatus() == jobPrepare {
					go func() {
						job.setStatus(jobRunning)
						f()
						if job.getStatus() == jobRunning {
							job.setStatus(jobPrepare)
						}
					}()
				}
			case <-job.close:
				timer.Stop()
				job.setStatus(jobCancel)
				return
			}
		}
	}()
	return job.JobId
}

func nextId() string {
	m := md5.New()
	now := time.Now()
	timeBytes, err := now.MarshalBinary()
	if err != nil {
		log.Fatalln(err)
	}
	m.Write(timeBytes)
	bs := m.Sum(nil)
	return hex.EncodeToString(bs)
}
