package main

import (
	"encoding/json"
	"sync"
)

type JobID uint64
type ClientID uint64

type JobManager struct {
	sync.Mutex
	Jobs       map[JobID]*Job
	Queues     map[string]*PriorityQueue[*Job]
	InProgress map[ClientID][]*Job
}

func (m *JobManager) AddJob(queue string, priority int, payload json.RawMessage) JobID {
	m.Lock()
	defer m.Unlock()

	job := &Job{
		ID:       GetNextJobId(),
		Queue:    queue,
		Priority: priority,
		Payload:  payload,
	}
	m.Jobs[job.ID] = job

	q := m.Queues[queue]
	if q == nil {
		q = new(PriorityQueue[*Job])
		m.Queues[queue] = q
	}
	q.Push(job, -job.Priority) // -priority for max heap

	return job.ID
}

func (m *JobManager) GetJob(id ClientID, queues []string) (Job, bool) {
	m.Lock()
	defer m.Unlock()

	var bestQueue *PriorityQueue[*Job]
	var bestPriority = -1
	for _, queue := range queues {
		q := m.Queues[queue]
		for q != nil && !q.Empty() && q.Peek().Deleted {
			q.Pop()
		}

		if q == nil || q.Empty() {
			continue
		}

		job := q.Peek()
		if job.Priority > bestPriority {
			bestQueue = q
			bestPriority = job.Priority
		}
	}

	if bestPriority < 0 {
		return Job{}, false
	}

	job := bestQueue.Pop()
	m.InProgress[id] = append(m.InProgress[id], job)
	return *job, true
}

func (m *JobManager) DeleteJob(id JobID) bool {
	m.Lock()
	defer m.Unlock()

	job := m.Jobs[id]
	if job == nil || job.Deleted {
		return false
	}

	job.Deleted = true
	delete(m.Jobs, id)
	return true
}

func (m *JobManager) AbortJob(clientID ClientID, jobID JobID) bool {
	m.Lock()
	defer m.Unlock()

	job := m.Jobs[jobID]
	if job == nil || job.Deleted {
		return false
	}

	for index, job := range m.InProgress[clientID] {
		if job.ID == jobID {
			// This client is no longer working on it
			m.InProgress[clientID][index] = m.InProgress[clientID][0]
			m.InProgress[clientID] = m.InProgress[clientID][1:]

			// Put it back into its queue
			m.Queues[job.Queue].Push(job, -job.Priority)

			return true
		}
	}

	return false
}

func (m *JobManager) OnClientDisconnect(id ClientID) {
	m.Lock()
	defer m.Unlock()

	for _, job := range m.InProgress[id] {
		if !job.Deleted {
			m.Queues[job.Queue].Push(job, -job.Priority)
		}
	}

	delete(m.InProgress, id)
}

var NextJobId JobID

func GetNextJobId() JobID {
	NextJobId++
	return NextJobId
}

type Job struct {
	ID       JobID
	Queue    string
	Priority int
	Payload  json.RawMessage
	Deleted  bool
}
