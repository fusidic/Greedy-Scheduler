package greedy

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/features"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

// implement the interfaces in scheduler framework
var (
	// _ framework.QueueSortPlugin = &Greedy{}
	_ framework.PreFilterPlugin = &Greedy{}
	_ framework.FilterPlugin    = &Greedy{}
	// _ framework.PostFilterPlugin = &Greedy{}
	_ framework.ScorePlugin     = &Greedy{}
	_ framework.ScoreExtensions = &Greedy{}

	scheme = runtime.NewScheme()
)

const (
	// Name ...
	Name              = "greedy"
	preFilterStateKey = "PreFilter" + Name
)

// Args ...
type Args struct {
	KubeConfig string `json:"kbueconfig,omitempty"`
	Master     string `json:"master,omitempty"`
}

// Greedy ...
type Greedy struct {
	args   *Args
	handle framework.FrameworkHandle
	resourceAllocationScorer
}

// Name ...
func (g Greedy) Name() string {
	return Name
}

// New initializes a new plugin and return it.
// And New should implement the function as frameworkruntime.PluginFactory do.
func New(configuration runtime.Object, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	// I HAVE NO IDEA HOW THESE CODES WORK.
	if err := frameworkruntime.DecodeInto(configuration, args); err != nil {
		return nil, err
	}

	resToWeightMap := make(resourceToWeightMap)
	resToWeightMap["cpu"] = 1
	resToWeightMap["memory"] = 1

	klog.V(3).Infof("get plugin config args: %+v", args)
	return &Greedy{
		args:   args,
		handle: f,
		resourceAllocationScorer: resourceAllocationScorer{
			Name: "NodeResourcesMostAllocated",
			// how can the compiler know it need greedyResourceScorer or the function it return with.
			scorer:              greedyResourceScorer(resToWeightMap),
			resourceToWeightMap: resToWeightMap,
		},
	}, nil

}

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	framework.Resource
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (g *Greedy) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// PreFilter invoked at the prefilter extension point.
func (g *Greedy) PreFilter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod) *framework.Status {
	// cycleState provides a mechanism for plugins to store and retrieve arbitrary data, with the form as "key to value"
	// preFilterStateKey is that Key to our value -- pod requested resources.
	cycleState.Write(preFilterStateKey, computePodResourceRequest(pod))
	// nil equals to framework.NewStatus(framework.Success, "")
	return nil
}

// computePodResourceRequest return a framework.Resource that covers the LARGEST width in each resource dimension.
func computePodResourceRequest(pod *v1.Pod) *preFilterState {
	result := &preFilterState{}
	for _, container := range pod.Spec.Containers {
		result.Add(container.Resources.Requests)
	}

	// take max_resource(sum_pod, any_init_container)
	for _, container := range pod.Spec.InitContainers {
		result.SetMaxResource(container.Resources.Requests)
	}

	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(features.PodOverhead) {
		result.Add(pod.Spec.Overhead)
	}

	return result
}

func getPrefilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState dowsn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %v", preFilterStateKey, err)
	}
	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to NodeResourcesFit.preFilterState error", c)
	}
	return s, nil
}

// Filter invoked at the filter extension point.
// Checks if a node has sufficient resources, such as cpu, memory, gpu, opaque int resources etc to run a pod.
// It returns a list of insufficient resources, if empty, then the node has all the resources requested by the pod.
func (g *Greedy) Filter(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// TODO: Filter
	klog.V(3).Infof("filter pod: %v, node: %v", pod.Name, nodeInfo.Node().Name)
	s, err := getPrefilterState(cycleState)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	insufficientResources := fitsRequest(s, nodeInfo)

	if len(insufficientResources) != 0 {
		// keep all failure reasons.
		failureReasons := make([]string, 0, len(insufficientResources))
		for _, r := range insufficientResources {
			failureReasons = append(failureReasons, r.Reason)
		}
		return framework.NewStatus(framework.Unschedulable, failureReasons...)
	}
	// return nil also mean that.
	return framework.NewStatus(framework.Success, "")
}

// InsufficientResource describes what kind of resource limit is hit and caused the pod to not fit the node.
type InsufficientResource struct {
	ResourceName v1.ResourceName
	// explicitly pass a parameter for reason to avoid formatting messages.
	Reason    string
	Requested int64
	Used      int64
	Capacity  int64
}

// Fits checks if node have enough resources to host the pod.
func Fits(pod *v1.Pod, nodeInfo *framework.NodeInfo) []InsufficientResource {
	return fitsRequest(computePodResourceRequest(pod), nodeInfo)
}

// fitsRequest returns requested resource's status and reason why node can not fit.
func fitsRequest(podRequest *preFilterState, nodeInfo *framework.NodeInfo) []InsufficientResource {
	insufficientResources := make([]InsufficientResource, 0, 4)

	allowedPodNumber := nodeInfo.Allocatable.AllowedPodNumber
	if len(nodeInfo.Pods)+1 > allowedPodNumber {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourcePods,
			"Too many pods",
			1,
			int64(len(nodeInfo.Pods)),
			int64(allowedPodNumber),
		})
	}

	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {
		return insufficientResources
	}

	if nodeInfo.Allocatable.MilliCPU < podRequest.MilliCPU+nodeInfo.Requested.MilliCPU {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceCPU,
			"Insufficient cpu",
			podRequest.MilliCPU,
			nodeInfo.Requested.MilliCPU,
			nodeInfo.Allocatable.MilliCPU,
		})
	}
	if nodeInfo.Allocatable.EphemeralStorage < podRequest.EphemeralStorage+nodeInfo.Requested.EphemeralStorage {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceEphemeralStorage,
			"Insufficient ephemeral-storage",
			podRequest.EphemeralStorage,
			nodeInfo.Requested.EphemeralStorage,
			nodeInfo.Allocatable.EphemeralStorage,
		})
	}
	if nodeInfo.Allocatable.Memory < podRequest.Memory+nodeInfo.Requested.Memory {
		insufficientResources = append(insufficientResources, InsufficientResource{
			v1.ResourceMemory,
			"Insufficient memory",
			podRequest.Memory,
			nodeInfo.Requested.Memory,
			nodeInfo.Allocatable.Memory,
		})
	}

	// leave for extension resources

	return insufficientResources
}

// Score rank nodes that passed the filtering phase, and it is invoked at the Score extension point.
func (g *Greedy) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	// TODO: Score
	nodeInfo, err := g.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil || nodeInfo.Node() == nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v, node is nil: %v", nodeName, err, nodeInfo.Node() == nil))
	}

	// g.score favors nodes with most requested resources.
	// It calculates the percentage of memory and CPU requested by pods scheduled on the node,
	// and prioritizes based on the maximum of the average of the fraction of requested to capacity.
	// Details: (cpu(MaxNodeScore * sum(requested) / capacity) + memory(MaxNodeScore * sum(requested) / capacity)) / weightSum
	// func score is from the delegation of resourceAllocationScorer
	return g.score(pod, nodeInfo)
}

// greddyResourceScorer is the custom scorer implement by users.
// it is defined in resourceAllocationScorer as a function member, and also as a callback in g.score
func greedyResourceScorer(resToWeightMap resourceToWeightMap) func(requested, allocable resourceToValueMap, includeVolumes bool, requestedVolumes int, allocatableVolumes int) int64 {
	return func(requested, allocable resourceToValueMap, includeVolumes bool, requestedVolumes int, allocatableVolumes int) int64 {
		var nodeScore, weightSum int64
		for resource, weight := range resToWeightMap {
			resourceScore := greedyScore(requested[resource], allocable[resource])
			nodeScore += resourceScore * weight
			weightSum += weight
		}
		return (nodeScore / weightSum)
	}
}

func greedyScore(requested, capacity int64) int64 {
	if capacity == 0 {
		return 0
	}
	if requested > capacity {
		return 0
	}
	return (requested * framework.MaxNodeScore) / capacity
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
