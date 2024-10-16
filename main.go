package main

import (
	"cube/worker"
)

//TIP To run your code, right-click the code and select <b>Run</b>. Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.

func main() {

	//fmt.Printf("create a test container\n")
	//dockerTask, createResult := task.CreateContainer()
	//
	//time.Sleep(time.Second * 5)
	//
	//fmt.Printf("stopping container %s\n", createResult.ContainerId)
	//_ = task.StopContainer(dockerTask, createResult.ContainerId)

	worker.Test_worker_with_state_transition()
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
