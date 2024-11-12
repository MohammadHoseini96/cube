package worker

import (
	"cube/stats"
	"cube/task"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	if err := d.Decode(&te); err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf(msg)
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HttpStatusCode: http.StatusBadRequest,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task: %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) InspectTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(400)
	}

	tID, _ := uuid.Parse(taskID)
	t, err := a.Worker.Db.Get(tID.String())
	if err != nil {
		log.Printf("No task with ID %v found", tID)
		w.WriteHeader(404)
		return
	}

	resp := a.Worker.InspectTask(t.(task.Task))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(resp.Container)

}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		log.Printf("No taskID passed in the request.\n")
		w.WriteHeader(http.StatusBadRequest)
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		log.Printf("Error parsing taskID: %s %v\n", taskID, err)
		w.WriteHeader(http.StatusBadRequest)
	}

	t, err := a.Worker.Db.Get(tID.String())
	if err != nil {
		log.Printf("No task with ID %v found", tID)
		w.WriteHeader(http.StatusNotFound)
	}

	// copy the task t so we change the state
	taskToStop := *t.(*task.Task)
	taskToStop.State = task.Completed
	a.Worker.AddTask(taskToStop)

	log.Printf("Added task %v to stop container %v\n", taskToStop.ID.String(), taskToStop.ContainerID)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}

func (a *Api) GetCpuUsageHandler(w http.ResponseWriter, r *http.Request) {
	interval, _ := strconv.Atoi(chi.URLParam(r, "interval"))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	usage := stats.CPUUsage{
		Percentage: stats.CpuUsagePercent(interval),
	}
	json.NewEncoder(w).Encode(&usage)
}
