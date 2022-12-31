package main

import (
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
)

type JobID uint64
type ClientID uint64

type JobManager struct {
	sync.Mutex
	Queues     map[string][]Job
	InProgress map[ClientID][]Job
}

func (m *JobManager) AddJob(queue string, priority int, payload json.RawMessage) JobID {
	m.Lock()
	defer m.Unlock()

	id := GetNextJobId()

	m.Queues[queue] = append(m.Queues[queue], Job{
		ID:       id,
		Queue:    queue,
		Priority: priority,
		Payload:  payload,
	})
	SortJobs(m.Queues[queue])

	return id
}

func (m *JobManager) GetJob(id ClientID, queues []string) (Job, bool) {
	m.Lock()
	defer m.Unlock()

	var bestQueue string
	var bestPriority = -1
	for _, queue := range queues {
		if len(m.Queues[queue]) == 0 {
			continue
		}

		if job := m.Queues[queue][0]; job.Priority > bestPriority {
			bestQueue = queue
			bestPriority = job.Priority
		}
	}

	var job Job
	if bestPriority < 0 {
		return job, false
	}

	job, m.Queues[bestQueue] = m.Queues[bestQueue][0], m.Queues[bestQueue][1:]
	m.InProgress[id] = append(m.InProgress[id], job)
	return job, true
}

func (m *JobManager) DeleteJob(id JobID) bool {
	m.Lock()
	defer m.Unlock()

	// First look for it in a queue
	for queue, jobs := range m.Queues {
		for index, job := range jobs {
			if job.ID == id {
				m.Queues[queue][index] = m.Queues[queue][0]
				m.Queues[queue] = m.Queues[queue][1:]
				SortJobs(m.Queues[queue])
				return true
			}
		}
	}

	// Next look for it in the in progress jobs
	for clientID, jobs := range m.InProgress {
		for index, job := range jobs {
			if job.ID == id {
				m.InProgress[clientID][index] = m.InProgress[clientID][0]
				m.InProgress[clientID] = m.InProgress[clientID][1:]
				return true
			}
		}
	}

	return false
}

func (m *JobManager) AbortJob(clientID ClientID, jobID JobID) bool {
	m.Lock()
	defer m.Unlock()

	// Happy path first, this job was assigned to this client
	for index, job := range m.InProgress[clientID] {
		if job.ID == jobID {
			// This client is no longer working on it
			m.InProgress[clientID][index] = m.InProgress[clientID][0]
			m.InProgress[clientID] = m.InProgress[clientID][1:]

			// Put it back into its queue
			m.Queues[job.Queue] = append(m.Queues[job.Queue], job)
			SortJobs(m.Queues[job.Queue])

			return true
		}
	}

	return false
}

func (m *JobManager) OnClientDisconnect(id ClientID) {
	m.Lock()
	defer m.Unlock()

	for _, job := range m.InProgress[id] {
		m.Queues[job.Queue] = append(m.Queues[job.Queue], job)
		SortJobs(m.Queues[job.Queue])
	}

	delete(m.InProgress, id)
}

var NextJobId atomic.Uint64

func GetNextJobId() JobID {
	return JobID(NextJobId.Add(1))
}

type Job struct {
	ID       JobID
	Queue    string
	Priority int
	Payload  json.RawMessage
}

func SortJobs(jobs []Job) {
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].Priority > jobs[j].Priority
	})
}
