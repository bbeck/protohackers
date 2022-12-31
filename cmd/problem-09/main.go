package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"io"
	"net"
	"sync/atomic"
	"time"
)

func main() {
	manager := JobManager{
		Queues:     make(map[string][]Job),
		InProgress: make(map[ClientID][]Job),
	}

	internal.RunTCPServer(func(conn net.Conn) {
		defer conn.Close()

		clientID := GetNextClientID()
		defer manager.OnClientDisconnect(clientID)

		disconnected := false
		defer func() { disconnected = true }()

		r := bufio.NewReaderSize(conn, 1024*1024)
		for !disconnected {
			bs, _, err := r.ReadLine()
			if err != nil {
				return
			}

			var request Request
			if err := json.Unmarshal(bs, &request); err != nil || !request.IsValid() {
				fmt.Printf("clientID:%d invalid request:%s\n", clientID, bs)
				io.WriteString(conn, InvalidRequest)
				return
			}

			switch request.Type {
			case "put":
				id := manager.AddJob(*request.Queue, request.Priority, request.Job)
				io.WriteString(conn, fmt.Sprintf(`{"status":"ok","id":%d}`, id))

			case "get":
				queues := request.GetQueues()

				for {
					job, ok := manager.GetJob(clientID, queues)
					if !ok && request.Wait {
						time.Sleep(100 * time.Millisecond)
						if disconnected {
							break
						}
						continue
					}

					if !ok {
						io.WriteString(conn, NoJob)
						break
					}

					bs, _ := json.Marshal(GetResponse{
						Status:   "ok",
						ID:       fmt.Sprintf("%d", job.ID),
						Job:      job.Payload,
						Priority: job.Priority,
						Queue:    job.Queue,
					})
					io.WriteString(conn, fmt.Sprintf("%s", bs))
					break
				}

			case "delete":
				if !manager.DeleteJob(request.ID) {
					io.WriteString(conn, NoJob)
					continue
				}

				io.WriteString(conn, Ok)

			case "abort":
				if !manager.AbortJob(clientID, request.ID) {
					io.WriteString(conn, NoJob)
					continue
				}

				io.WriteString(conn, Ok)
			}
		}
	})
}

type Request struct {
	Type string `json:"request"`

	// Put
	Queue    *string         `json:"queue"`
	Priority int             `json:"pri"`
	Job      json.RawMessage `json:"job"`

	// Get
	Queues []*string `json:"queues"`
	Wait   bool      `json:"wait"`

	// Delete & Abort
	ID JobID `json:"id"`
}

func (r *Request) IsValid() bool {
	switch r.Type {
	case "put":
		return r.Queue != nil && r.Priority >= 0

	case "get":
		for _, queue := range r.Queues {
			if queue == nil {
				return false
			}
		}
		return true

	case "delete":
		return true

	case "abort":
		return true

	default:
		return false
	}
}

func (r *Request) GetQueues() []string {
	out := make([]string, len(r.Queues))
	for i := range r.Queues {
		out[i] = *r.Queues[i]
	}
	return out
}

type GetResponse struct {
	Status   string          `json:"status"`
	ID       string          `json:"id"`
	Job      json.RawMessage `json:"job"`
	Priority int             `json:"pri"`
	Queue    string          `json:"queue"`
}

var Ok = `{"status":"ok"}`
var InvalidRequest = `{"status":"error", "error":"invalid request"}`
var NoJob = `{"status":"no-job"}`

var NextClientID atomic.Uint64

func GetNextClientID() ClientID {
	return ClientID(NextClientID.Add(1))
}
