package register

import (
	"github.com/fusidic/Greedy-Scheduler/pkg/greedy"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

// Register custom plugins to kubernetes scheduler framework
func Register() *cobra.Command {
	return app.NewSchedulerCommand(
		app.WithPlugin(greedy.Name, greedy.New),
	)
}
