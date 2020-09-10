package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
	"k8s.io/component-base/logs"
	"github.com/fusidic/Greedy-Scheduler/pkg/register"	
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	command := register.Register()
	logs.InitLogs()

	defer logs.FlushLogs()
	if err := command.Excute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
