package greedy

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

// implement the interfaces in scheduler framework
var (
	_ framework.QueueSortPlugin = &Greedy{}
	_ framework.FilterPlugin    = &Greedy{}
	// _ framework.PostFilterPlugin = &Greedy{}
	_ framework.ScorePlugin     = &Greedy{}
	_ framework.ScoreExtensions = &Greedy{}

	scheme = runtime.NewScheme()
)

const (
	// Name ...
	Name = "greedy"
)

// Args ...
type Args struct {
	KubeConfig string `json:"kbueconfig.omitempty"`
	Master     string `json:"master,omitempty"`
}

// Greedy ...
type Greedy struct {
	args   *Args
	handle framework.FrameworkHandle
}

// Less implement the sorting function in QueueSortPlugin
func (g *Greedy) Less(podInfo1, podInfo2 *framework.QueuedPodInfo) bool {
	// TODO: return sort.Less(podInfo1, podInfo2)
	return true
}

// Filter implement Filter() function which is defined in FilterPlugin
func (g *Greedy) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// TODO: Filter
	return nil
}

// Score rank nodes that passed the filtering phase
func (g *Greedy) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	// TODO: Score
	return 0, nil
}

// ScoreExtensions is defined in interface ScorePlugin and
// return a ScoreExtensions interface if it has been implemented.
func (g *Greedy) ScoreExtensions() framework.ScoreExtensions {
	return g
}

// NormalizeScore is an interface that must be implemented by "Score" plugins to
// rank nodes that passed the filtering phase.
func (g *Greedy) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	// TODO: NormalizeScore
	return nil
}

//

// Name ...
func (g Greedy) Name() string {
	return Name
}

// New ...
// Should be same with frameworkruntime.PluginFatory
func New(configuration *runtime.Object, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	if err := frameworkruntime.DecodeInto(configuration, args); err != nil {
		return nil, err
	}
	klog.V(3).Infof("get plugin config args: %+v", args)
	return &Greedy{
		args:   args,
		handle: f,
	}, nil
}
