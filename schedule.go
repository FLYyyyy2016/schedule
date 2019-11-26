package schedule

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

const (
	JobPrepare JobStatus = iota
	JobRunning
	JobFinish
	JobCreating
	JobCancel
	JobNotExist
)

type Schedule struct {
	tasks map[string]Task
	sync.Mutex
}

type JobStats struct {
	JobStatus    JobStatus
	FinishedTime int
}

type JobStatus uint

type JobIsRunningError struct{}

func (error JobIsRunningError) Error() string {
	return "jobBase is running ,you can try it again"
}

type JobIsCreatingError struct{}

func (error JobIsCreatingError) Error() string {
	return "jobBase is Creating ,you can try it again"
}

type JobIsCancelError struct{}

func (error JobIsCancelError) Error() string {
	return "jobBase is canceled "
}

type JobIsFinishError struct{}

func (error JobIsFinishError) Error() string {
	return "jobBase is finished"
}

type JobNotExistError struct{}

func (error JobNotExistError) Error() string {
	return "jobBase not exist ,you can try it again"
}

func NewSchedule() *Schedule {
	return &Schedule{tasks: make(map[string]Task)}
}

func (sche *Schedule) Delay(duration time.Duration) Task {
	sche.Lock()
	defer sche.Unlock()
	newJob := DelayJob{
		jobBase: jobBase{JobId: nextId(), close: make(chan struct{}), duration: duration},
	}
	newJob.setStatus(JobCreating)
	sche.tasks[newJob.JobId] = &newJob
	return &newJob
}

func (sche *Schedule) Every(duration time.Duration) Task {
	sche.Lock()
	defer sche.Unlock()
	newJob := EveryJob{
		jobBase: jobBase{JobId: nextId(), close: make(chan struct{}), duration: duration},
	}
	newJob.setStatus(JobCreating)
	sche.tasks[newJob.JobId] = &newJob
	return &newJob
}

func (sche *Schedule) Query(jobId string) (JobStats, error) {
	sche.Lock()
	defer sche.Unlock()
	if task, ok := sche.tasks[jobId]; ok {
		return task.getJobStats(), nil
	} else {
		return JobStats{}, JobNotExistError{}
	}
}

func (sche *Schedule) Cancel(jobId string) error {
	sche.Lock()
	defer sche.Unlock()
	if task, ok := sche.tasks[jobId]; ok {
		return task.cancel()
	} else {
		return JobNotExistError{}
	}
}

type Task interface {
	Do(func()) string
	getId() string
	cancel() error
	getJobStats() JobStats
}

type DelayJob struct {
	jobBase
}

type EveryJob struct {
	jobBase
}

type jobBase struct {
	sync.Mutex
	JobId     string
	status    JobStatus
	isCreated bool
	close     chan struct{}
	duration  time.Duration
	finish    int
}

func (job *jobBase) haveRun() bool {
	job.Lock()
	defer job.Unlock()
	if !job.isCreated {
		job.isCreated = true
		return true
	} else {
		return false
	}
}

func (job *jobBase) setStatus(status JobStatus) {
	job.Lock()
	defer job.Unlock()
	job.status = status
}
func (job *jobBase) getStatus() JobStatus {
	job.Lock()
	defer job.Unlock()
	return job.status
}

func (job *jobBase) changeStatus(result, target JobStatus) bool {
	job.Lock()
	defer job.Unlock()
	if job.status == result {
		job.status = target
		return true
	} else {
		return false
	}
}

func (job *jobBase) getJobStats() JobStats {
	job.Lock()
	defer job.Unlock()
	return JobStats{
		JobStatus:    job.status,
		FinishedTime: job.finish,
	}
}

func (job *DelayJob) cancel() error {
	status := job.getStatus()
	switch status {
	case JobPrepare:
		job.close <- struct{}{}
	case JobRunning:
		return JobIsRunningError{}
	case JobCancel:
		return nil
	case JobCreating:
		return JobIsCreatingError{}
	case JobFinish:
		return JobIsFinishError{}
	}
	return nil
}

func (job *EveryJob) cancel() error {
	status := job.getStatus()
	switch status {
	case JobPrepare:
		job.close <- struct{}{}
	case JobRunning:
		job.close <- struct{}{}
	case JobCancel:
		return nil
	case JobCreating:
		return JobIsCreatingError{}
	case JobFinish:
		return JobIsFinishError{}
	}
	return nil
}

func (job *jobBase) getId() string {
	return job.JobId
}

func (job *DelayJob) Do(f func()) string {
	if job.haveRun() {
		return job.JobId
	}
	timer := time.NewTimer(job.duration)
	go func() {
		job.setStatus(JobPrepare)
		select {
		case <-timer.C:
			if job.changeStatus(JobPrepare, JobRunning) {
				go func() {
					f()
					job.finishOneTime()
					job.changeStatus(JobRunning, JobFinish)
				}()
			}
		case <-job.close:
			timer.Stop()
			job.setStatus(JobCancel)
			return
		}
	}()
	return job.JobId
}

func (job *jobBase) finishOneTime() {
	job.Lock()
	defer job.Unlock()
	job.finish++
}

func (job *EveryJob) Do(f func()) string {
	if !job.haveRun() {
		return job.JobId
	}
	timer := time.NewTicker(job.duration)
	go func() {
		job.setStatus(JobPrepare)
		for {
			select {
			case <-timer.C:
				if job.changeStatus(JobPrepare, JobRunning) {
					go func() {
						f()
						job.finishOneTime()
						job.changeStatus(JobRunning, JobPrepare)
					}()
				}
			case <-job.close:
				timer.Stop()
				job.setStatus(JobCancel)
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
