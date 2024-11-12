package manager

import (
	"bytes"
	"cube/node"
	"cube/scheduler"
	"cube/store"
	"cube/task"
	"cube/worker"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        store.Store
	EventDb       store.Store
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
	WorkerNodes   []*node.Node
	Scheduler     scheduler.Scheduler
}

func New(workers []string, schedulerType string, dbType string) *Manager {

	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	var nodes []*node.Node
	for w := range workers {
		workerTaskMap[workers[w]] = []uuid.UUID{}

		nodeAPI := fmt.Sprintf("http://%v", workers[w])
		n := node.NewNode(workers[w], nodeAPI, "worker")
		nodes = append(nodes, n)
	}

	var s scheduler.Scheduler
	switch schedulerType {
	case "roundrobin":
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	case "epvm":
		s = &scheduler.Epvm{Name: "epvm"}
	default:
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	}

	m := Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}

	var ts store.Store
	var es store.Store
	var err error

	switch dbType {
	case "memory":
		ts = store.NewInMemoryTaskStore()
		es = store.NewInMemoryTaskEventStore()
	case "persistent":
		ts, err = store.NewTaskStore("tasks.db", 0600, "tasks")
		if err != nil {
			log.Fatalf("unable to create task store: %v", err)
		}
		es, err = store.NewEventStore("events.db", 0600, "events")
		if err != nil {
			log.Fatalf("unable to create task event store: %v", err)
		}
	}

	m.TaskDb = ts
	m.EventDb = es
	return &m
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil {
		return nil, errors.New(fmt.Sprintf("No availabe candidates match resource request for task %v", t.ID))

	}
	scores := m.Scheduler.Score(t, candidates)
	if scores == nil {
		return nil, errors.New(fmt.Sprintf("No scores returned to task %v", t))
	}
	selectedNode := m.Scheduler.Pick(scores, candidates)

	return selectedNode, nil
}

func (m *Manager) UpdateTasks() {
	for {
		log.Printf("Checking for task updates from workers")
		m.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

func (m *Manager) updateTasks() {
	for _, w := range m.Workers {
		log.Printf("[manager] Checking worker %v for task updates", w)
		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[manager] Error connecting to %v: %v", w, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[manager] Error sending request %v\n", err)
			continue
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		if err := d.Decode(&tasks); err != nil {
			log.Printf("[manager] Error unmarshaling tasks %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("[manager] Attemting to update task %v\n", t.ID)

			result, err := m.TaskDb.Get(t.ID.String())
			if err != nil {
				log.Printf("[manager] %s\n", err)
				continue
			}

			taskPersisted, ok := result.(*task.Task)
			if !ok {
				log.Printf("[manager] cannot convert result %v to task.Task type\n", result)
				continue
			}

			if taskPersisted.State != t.State {
				taskPersisted.State = t.State
			}
			taskPersisted.StartTime = t.StartTime
			taskPersisted.FinishTime = t.FinishTime
			taskPersisted.ContainerID = t.ContainerID
			taskPersisted.HostPorts = t.HostPorts

			m.TaskDb.Put(taskPersisted.ID.String(), taskPersisted)
		}
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Printf("Processing any tasks in the queue")
		m.SendWork()
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) stopTask(worker string, taskID string) {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("Error creating request to delete task: %s: %v\n", taskID, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error connecting to worker at %s: %v\n", url, err)
		return
	}

	if resp.StatusCode != http.StatusNoContent {
		log.Printf("Error sending request to delete task %v\n", err)
		return
	}

	log.Printf("task %s has been scheduled to be stopped\n", taskID)
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		e := m.Pending.Dequeue()
		te := e.(task.TaskEvent)
		if err := m.EventDb.Put(te.ID.String(), &te); err != nil {
			log.Printf("error attempting to store task event %s: %s\n", te.ID.String(), err)
		}
		log.Printf("Pulled %v off pending queue\n", te)

		taskWorker, ok := m.TaskWorkerMap[te.Task.ID]
		if ok {
			result, err := m.TaskDb.Get(te.Task.ID.String())
			if err != nil {
				log.Printf("unable to schedule task: %s\n", err)
				return
			}
			persistedTask, ok := result.(*task.Task)
			if !ok {
				log.Printf("unable to convert task to task.Task type\n")
				return
			}

			if te.State == task.Completed && task.ValidateStateTransition(persistedTask.State, te.State) {
				m.stopTask(taskWorker, te.Task.ID.String())
				return
			}
			log.Printf(
				"invalid request: existing task %s is in state %v and cannot transition to the completed state",
				persistedTask.ID.String(),
				persistedTask.State,
			)
			return
		}

		t := te.Task
		w, err := m.SelectWorker(t)
		if err != nil {
			log.Printf("Error selecting worker %s for task: %v\n", t.ID, err)
			return
		}
		m.WorkerTaskMap[w.Name] = append(m.WorkerTaskMap[w.Name], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w.Name

		t.State = task.Scheduled
		m.TaskDb.Put(t.ID.String(), &t)

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("Unable to marshal task object: %v.\n", err)
		}

		url := fmt.Sprintf("http://%s/tasks", w.Name)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error connecting to %v: %v.\n", w.Name, err)
			m.Pending.Enqueue(te)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			errResponse := worker.ErrResponse{}
			errResponseDecode := d.Decode(&errResponse)
			if errResponseDecode != nil {
				fmt.Printf("Error decoding response: %v\n", errResponseDecode.Error())
				return
			}
			log.Printf("Resonse error (%d): %s", errResponse.HttpStatusCode, errResponse.Message)
			return
		}

		t = task.Task{}
		if err = d.Decode(&t); err != nil {
			fmt.Printf("Error decoding response: %s.\n", err.Error())
			return
		}
		log.Printf("put task: %v in the worker %s\n", t, w.Name)
	} else {
		log.Println("No work in the queue")
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	taskList, err := m.TaskDb.List()
	if err != nil {
		log.Printf("Error getting list of tasks: %v\n", err)
		return nil
	}
	return taskList.([]*task.Task)
}

func (m *Manager) checkTasksHealth(t task.Task) error {
	log.Printf("Calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	wrkr := strings.Split(w, ":")
	if hostPort == nil {
		log.Printf("Have not collected task %s host port yest. Skipping.\n", t.ID)
		return nil
	}
	url := fmt.Sprintf("http://%s:%s%s", wrkr[0], *hostPort, t.HealthCheck)
	log.Printf("Calling health check for task %s: %s\n", t.ID, url)
	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check %s\n", url)
		log.Printf(msg)
		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.ID)
		log.Printf(msg)
		return errors.New(msg)
	}

	log.Printf("Task %s health check response %v\n", t.ID, resp.StatusCode)
	return nil
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < 3 {
			if err := m.checkTasksHealth(*t); err != nil {
				//if t.RestartCount < 3 {
				//	m.restartTask(t)
				//}
				m.restartTask(t)
			}
		} else if t.State == task.Failed && t.RestartCount < 3 {
			m.restartTask(t)
		}
	}
}

func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Performing task health check")
		m.doHealthChecks()
		log.Println("Task health check completed")
		log.Println("Sleeping for 40 seconds")
		time.Sleep(35 * time.Second)
	}
}

func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb.Put(t.ID.String(), t)

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}
	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Unable to marshal task object: %v.\n", err)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v: %v.\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		e := worker.ErrResponse{}
		err := d.Decode(&e)
		if err != nil {
			log.Printf("Error decoding response: %v.\n", err)
			return
		}
		log.Printf("Resonse error (%d): %s", e.HttpStatusCode, e.Message)
		return
	}

	restartedTask := task.Task{}
	if err = d.Decode(&restartedTask); err != nil {
		fmt.Printf("Error decoding response: %v.\n", err.Error())
		return
	}
	log.Printf("Task %v was set to Scheduled state and sent via POST request. \n", t)
}
