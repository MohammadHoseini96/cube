package manager

import (
	"cube/task"
	"cube/worker"
	"fmt"
	"github.com/google/uuid"
	"time"
)

func Serve_manager(host string, port int, numTask int) {
	go worker.Serve_worker_with_api(host, port)

	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := New(workers)

	for i := 0; i < numTask; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}
		m.AddTask(te)
		m.SendWork()
	}

	go func() {
		for {
			fmt.Printf("[Manager] Updating tasks from %d workers\n", len(m.Workers))
			m.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDb {
			fmt.Printf("[Manager] Task id: %s, state: %d\n", t.ID, t.State)
			time.Sleep(15 * time.Second)
		}
	}

}
