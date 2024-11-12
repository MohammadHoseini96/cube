package manager

import (
	"cube/task"
	"cube/worker"
	"fmt"
	"github.com/google/uuid"
	"time"
)

func ServeManager(host string, port int, numTask int) {
	go worker.ServeWorkerWithApi(host, port)

	fmt.Println("Starting Cube manager")
	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := New(workers, "roundrobin", "memory")

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

	//go func() {
	//	for {
	//		fmt.Printf("[Manager] Updating tasks from %d workers\n", len(m.Workers))
	//		m.UpdateTasks()
	//		time.Sleep(15 * time.Second)
	//	}
	//}()
	m.UpdateTasks()

	for {
		taskList, _ := m.TaskDb.List()
		tasks := taskList.([]task.Task)
		for _, t := range tasks {
			fmt.Printf("[Manager] Task id: %s, state: %d\n", t.ID, t.State)
			time.Sleep(15 * time.Second)
		}
	}
}

func ServeManagerWithApi(managerHost string, managerPort int, workerHost string, workerPort int) {
	go worker.ServeWorkerWithApi(workerHost, workerPort)

	fmt.Println("Starting Cube manager")
	workers := []string{fmt.Sprintf("%s:%d", workerHost, workerPort)}
	m := New(workers, "roundrobin", "memory")
	managerApi := Api{
		Address: managerHost,
		Port:    managerPort,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	managerApi.Star()
}

func ServeManagerWithMultipleWorkers(managerHost string, managerPort int, workersMap map[int]string, dbType string) {
	go worker.ServeWorkersWithApi(workersMap, dbType)

	fmt.Println("Starting Cube manager")

	var workers []string
	for workerPort, workerHost := range workersMap {
		workers = append(workers, fmt.Sprintf("%s:%d", workerHost, workerPort))
	}
	m := New(workers, "epvm", dbType)
	managerApi := Api{
		Address: managerHost,
		Port:    managerPort,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	managerApi.Star()
}
