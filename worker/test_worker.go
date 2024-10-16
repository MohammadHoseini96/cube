package worker

import (
	"cube/task"
	"fmt"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"time"
)

func Test_worker_with_state_transition() {
	db := make(map[uuid.UUID]*task.Task)
	w := Worker{
		Name:  "test-worker-1",
		Db:    db,
		Queue: *queue.New(),
	}

	t := task.Task{
		ID:    uuid.New(),
		Name:  "test-container-1",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}

	fmt.Printf("starting the task: %v\n", t)
	w.AddTast(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error.Error())
	}

	t.ContainerID = result.ContainerId
	fmt.Printf("task %s is running in container: %s\n", t.ID, t.ContainerID)
	fmt.Println("Sleep time!")
	time.Sleep(5 * time.Second)

	fmt.Printf("stopping task : %s\n", t.ID)
	t.State = task.Completed
	w.AddTast(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error.Error())
	}
	fmt.Printf("task %s is completeed: %s\n", t.Name, result.Result)
	//todo: use assert functions
}
