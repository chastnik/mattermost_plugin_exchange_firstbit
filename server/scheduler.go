package main

import (
	"sync"
	"time"
)

// Job represents a scheduled job
type Job struct {
	Name     string
	Interval time.Duration
	Task     func()
	ticker   *time.Ticker
	quit     chan bool
}

// Scheduler manages periodic tasks
type Scheduler struct {
	plugin *Plugin
	jobs   map[string]*Job
	mutex  sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(plugin *Plugin) *Scheduler {
	return &Scheduler{
		plugin: plugin,
		jobs:   make(map[string]*Job),
	}
}

// AddJob adds a new periodic job
func (s *Scheduler) AddJob(name string, interval time.Duration, task func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Stop existing job if it exists
	if existingJob, exists := s.jobs[name]; exists {
		s.stopJob(existingJob)
	}

	job := &Job{
		Name:     name,
		Interval: interval,
		Task:     task,
		ticker:   time.NewTicker(interval),
		quit:     make(chan bool),
	}

	s.jobs[name] = job
	go s.runJob(job)
}

// runJob runs a periodic job
func (s *Scheduler) runJob(job *Job) {
	for {
		select {
		case <-job.ticker.C:
			job.Task()
		case <-job.quit:
			job.ticker.Stop()
			return
		}
	}
}

// RemoveJob removes a job
func (s *Scheduler) RemoveJob(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if job, exists := s.jobs[name]; exists {
		s.stopJob(job)
		delete(s.jobs, name)
	}
}

// stopJob stops a running job
func (s *Scheduler) stopJob(job *Job) {
	job.quit <- true
	job.ticker.Stop()
}

// Stop stops all jobs
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, job := range s.jobs {
		s.stopJob(job)
	}
	s.jobs = make(map[string]*Job)
}
