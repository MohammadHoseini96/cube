package worker

import (
	"cube/task"
	"errors"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"log"
	"time"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func (w *Worker) GetTasks() []*task.Task {
	var tasks []*task.Task
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}

func (w *Worker) CollectStats() {
	for {
		log.Println("Collecting stats")
		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("Error running task: %v", result.Error)
			}
		} else {
			log.Println("No task to process currently.")
		}
		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks in the queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)

	taskPersisted := w.Db[taskQueued.ID]
	if taskPersisted == nil {
		w.Db[taskQueued.ID] = &taskQueued
		taskPersisted = &taskQueued
	}

	var result task.DockerResult
	if task.ValidateStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we should not get here")
		}
	} else {
		result.Error = fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
	}
	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now()
	conf := task.NewConfig(&t)
	d := task.NewDocker(conf)

	result := d.Run()
	if result.Error != nil {
		log.Printf("Error running task %v: %v\n", t.Name, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}
	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	conf := task.NewConfig(&t)
	d := task.NewDocker(conf)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf("Error stopping container %v: %v", t.ContainerID, result.Error)
		//t.State = task.Failed
		//w.Db[t.ID] = &t
		return result
	}
	t.FinishTime = time.Now()
	t.State = task.Completed
	w.Db[t.ID] = &t
	log.Printf("Stopped and removed container %v for task %v\n", t.ContainerID, t.ID)

	return result
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	conf := task.NewConfig(&t)
	d := task.NewDocker(conf)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds.")
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) updateTasks() {
	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("ERROR: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("No container for running task %v\n", id)
				w.Db[id].State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s in non-running state %s\n", id, resp.Container.State.Status)
				w.Db[id].State = task.Failed
			}

			//w.Db[id].HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
			w.Db[id].HostPorts = resp.Container.HostConfig.PortBindings
		}
	}
}
