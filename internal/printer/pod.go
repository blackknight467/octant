/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package printer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiEquality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/vmware-tanzu/octant/internal/link"
	"github.com/vmware-tanzu/octant/internal/log"
	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/internal/util/kubernetes"
	"github.com/vmware-tanzu/octant/pkg/store"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

var podResourceCols = component.NewTableCols("Container", "Request: Memory", "Request: CPU", "Limit: Memory", "Limit: CPU")

// PodListHandler is a printFunc that prints pods
func PodListHandler(ctx context.Context, list *corev1.PodList, opts Options) (component.Component, error) {
	if list == nil {
		return nil, errors.New("list is nil")
	}

	// Check metrics-server availability once before building the table.
	var pml *octant.ClusterPodMetricsLoader
	if cc := opts.DashConfig.ClusterClient(); cc != nil {
		if loader, err := octant.NewClusterPodMetricsLoader(cc); err == nil {
			if ok, err := loader.SupportsMetrics(ctx); err == nil && ok {
				pml = loader
			}
		}
	}

	colNames := []string{"Name", "Ready", "Status", "Restarts", "Node", "Age"}
	if pml != nil {
		colNames = []string{"Name", "Ready", "Status", "CPU / Memory", "Restarts", "Node", "Age"}
	}
	if !opts.DisableLabels {
		// Insert "Labels" after "Name"
		colNames = append([]string{colNames[0], "Labels"}, colNames[1:]...)
	}

	cols := component.NewTableCols(colNames...)
	ot := NewObjectTable("Pods", "We couldn't find any pods!", cols, opts.DashConfig.ObjectStore())
	ot.AddFilters(podTableFilters())
	ot.EnablePluginStatus(opts.DashConfig.PluginManager())

	for i := range list.Items {
		row := component.TableRow{}
		pod := list.Items[i]

		nameLink, err := opts.Link.ForObject(&pod, pod.Name)
		if err != nil {
			return nil, err
		}
		row["Name"] = nameLink

		if !opts.DisableLabels {
			row["Labels"] = component.NewLabels(pod.Labels)
		}

		readyCounter := 0
		for _, c := range pod.Status.ContainerStatuses {
			if c.Ready {
				readyCounter++
			}
		}
		row["Ready"] = component.NewText(fmt.Sprintf("%d/%d", readyCounter, len(pod.Spec.Containers)))

		row["Status"] = podCombinedStatusText(&pod)

		if pml != nil {
			row["CPU / Memory"] = podListResourceText(ctx, pml, &pod)
		}

		restartCounter := 0
		for _, c := range pod.Status.ContainerStatuses {
			restartCounter += int(c.RestartCount)
		}
		row["Restarts"] = component.NewText(fmt.Sprintf("%d", restartCounter))

		nodeComponent, err := podNode(&pod, opts.Link)
		if err != nil {
			return nil, err
		}
		row["Node"] = nodeComponent

		row["Age"] = component.NewTimestamp(pod.CreationTimestamp.Time)

		if err := ot.AddRowForObject(ctx, &pod, row); err != nil {
			return nil, fmt.Errorf("add row for object: %w", err)
		}
	}

	ot.SetSortOrder("Name", false)
	return ot.ToComponent()
}

// podCombinedStatusText returns a single text component for the combined
// Phase/Status column. The status reason is shown when available; phase is
// shown only when there is no more specific status (e.g. unscheduled Pending
// pods). Color encodes phase: red=Failed/Unknown, yellow=Pending.
// The phase is NOT appended as text so that filter matching on "Pending" and
// "Running" continues to work for pods whose status reason differs from phase.
func podCombinedStatusText(pod *corev1.Pod) *component.Text {
	statusStr := ""
	if len(pod.Status.ContainerStatuses) > 0 {
		last := pod.Status.ContainerStatuses[len(pod.Status.ContainerStatuses)-1]
		if last.State.Waiting != nil {
			statusStr = last.State.Waiting.Reason
		} else if last.State.Running != nil {
			statusStr = "Running"
		}
		if pod.DeletionTimestamp != nil && last.State.Terminated == nil {
			statusStr = "Terminating"
		}
	}

	display := statusStr
	if display == "" {
		display = string(pod.Status.Phase)
	}

	t := component.NewText(display)
	// The filter matches on phase so that "ContainerCreating" pods still match
	// the "Pending" filter, and "Running" pods match regardless of specific reason.
	t.SetFilterValue(string(pod.Status.Phase))
	switch pod.Status.Phase {
	case corev1.PodFailed, corev1.PodUnknown:
		t.SetClassName("text-metric-error")
	case corev1.PodPending:
		t.SetClassName("text-metric-warning")
	}
	return t
}

// podListResourceText loads metrics for a single pod and returns a "CPU / Memory"
// text component. Returns a dash if metrics are not yet available for the pod.
func podListResourceText(ctx context.Context, pml *octant.ClusterPodMetricsLoader, pod *corev1.Pod) *component.Text {
	metricsObj, found, err := pml.Load(ctx, pod.Namespace, pod.Name)
	if err != nil || !found {
		return component.NewText("-")
	}

	containersRaw, found, err := unstructured.NestedSlice(metricsObj.Object, "containers")
	if err != nil || !found {
		return component.NewText("-")
	}

	cpuUsage := resource.Quantity{}
	memUsage := resource.Quantity{}
	for _, c := range containersRaw {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		usage, found, err := unstructured.NestedMap(container, "usage")
		if err != nil || !found {
			continue
		}
		if q, err := podMetricQuantity(usage, "cpu"); err == nil {
			cpuUsage.Add(q)
		}
		if q, err := podMetricQuantity(usage, "memory"); err == nil {
			memUsage.Add(q)
		}
	}

	return component.NewText(fmt.Sprintf("%vm / %vMi",
		cpuUsage.MilliValue(),
		memUsage.Value()/(1024*1024),
	))
}

func podNode(pod *corev1.Pod, linkGenerator link.Interface) (component.Component, error) {
	if nodeName := pod.Spec.NodeName; nodeName != "" {
		return linkGenerator.ForGVK("", "v1", "Node", pod.Spec.NodeName, pod.Spec.NodeName)
	}

	return component.NewText("<not scheduled>"), nil
}

var podConditionColumns = [][]string{
	{"Type", "type"},
	{"Reason", "reason"},
	{"Status", "status"},
	{"Message", "message"},
	{"Last Probe", "lastProbeTime"},
	{"Last Transition", "lastTransitionTime"},
}

// PodHandler is a printFunc that prints Pods
func PodHandler(ctx context.Context, pod *corev1.Pod, options Options) (component.Component, error) {
	o := NewObject(pod)
	o.ConditionsGen = conditionsGenFactory("", podConditionColumns, nil)
	o.EnableEvents()

	ph, err := newPodHandler(pod, o)
	if err != nil {
		return nil, err
	}

	if err := ph.Config(options); err != nil {
		return nil, errors.Wrap(err, "print pod configuration")
	}
	if err := ph.Status(options); err != nil {
		return nil, errors.Wrap(err, "print pod status")
	}
	if err := ph.Metrics(ctx, options); err != nil {
		return nil, errors.Wrap(err, "print pod metrics")
	}
	if err := ph.InitContainers(ctx, options); err != nil {
		return nil, errors.Wrap(err, "print pod init containers")
	}
	if err := ph.Containers(ctx, options); err != nil {
		return nil, errors.Wrap(err, "print pod containers")
	}
	if err := ph.EphemeralContainers(ctx, options); err != nil {
		return nil, errors.Wrap(err, "print pod ephemeral containers")
	}
	if err := ph.Additional(options); err != nil {
		return nil, errors.Wrap(err, "print pod additional items")
	}

	return o.ToComponent(ctx, options)
}

func createPodSummaryStatus(pod *corev1.Pod) (*component.Summary, error) {
	if pod == nil {
		return nil, errors.New("pod is nil")
	}

	summary := component.NewSummary("Status")

	sections := component.SummarySections{}

	sections.AddText("QoS", string(pod.Status.QOSClass))

	if pod.DeletionTimestamp != nil {
		summary.SetAlert(component.NewAlert(component.AlertStatusError, component.AlertTypeDefault, "Pod is being deleted", false, nil))

		sections = append(sections, component.SummarySection{
			Header:  "Status: Terminating",
			Content: component.NewText(pod.DeletionTimestamp.String()),
		})
		if pod.DeletionGracePeriodSeconds != nil {
			sections.AddText("Termination Grace Period", fmt.Sprintf("%ds", *pod.DeletionGracePeriodSeconds))
		}
	} else {
		sections.AddText("Phase", string(pod.Status.Phase))
	}

	if pod.Status.Reason != "" {
		sections.AddText("Reason", pod.Status.Reason)
	}
	if pod.Status.Message != "" {
		sections.AddText("Message", pod.Status.Message)
	}

	sections.AddText("Pod IP", pod.Status.PodIP)
	sections.AddText("Host IP", pod.Status.HostIP)

	if pod.Status.NominatedNodeName != "" {
		sections.AddText("NominatedNodeName", pod.Status.NominatedNodeName)
	}

	summary.Add(sections...)

	return summary, nil
}

type podStatus struct {
	Running   int
	Waiting   int
	Succeeded int
	Failed    int
}

func createPodStatus(pods []*corev1.Pod) podStatus {
	var ps podStatus

	for _, pod := range pods {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			ps.Running++
		case corev1.PodPending:
			ps.Waiting++
		case corev1.PodSucceeded:
			ps.Succeeded++
		case corev1.PodFailed:
			ps.Failed++
		}
	}

	return ps
}

// PodConfiguration generates pod configuration.
type PodConfiguration struct {
	pod *corev1.Pod
}

// NewPodConfiguration creates an instance of PodConfiguration.
func NewPodConfiguration(p *corev1.Pod) *PodConfiguration {
	return &PodConfiguration{
		pod: p,
	}
}

// Create creates a pod configuration summary.
func (p *PodConfiguration) Create(options Options) (*component.Summary, error) {
	if p.pod == nil {
		return nil, errors.New("pod is nil")
	}
	pod := p.pod

	sections := component.SummarySections{}

	if pod.Spec.Priority != nil {
		sections.AddText("Priority", fmt.Sprintf("%d", *pod.Spec.Priority))
	}
	if pod.Spec.PriorityClassName != "" {
		sections.AddText("PriorityClassName", pod.Spec.PriorityClassName)
	}

	contentLink, err := options.Link.ForGVK(pod.Namespace, "v1", "ServiceAccount", pod.Spec.ServiceAccountName, pod.Spec.ServiceAccountName)
	if err != nil {
		return nil, err
	}

	nodeLink, err := podNode(p.pod, options.Link)
	if err != nil {
		return nil, err
	}
	sections.Add("Node", nodeLink)

	sections = append(sections, component.SummarySection{
		Header:  "Service Account",
		Content: contentLink,
	})

	summary := component.NewSummary("Configuration", sections...)
	return summary, nil
}

func listPods(ctx context.Context, namespace string, selector *metav1.LabelSelector, uid types.UID, o store.Store) ([]*corev1.Pod, error) {
	key := store.Key{
		Namespace:  namespace,
		APIVersion: "v1",
		Kind:       "Pod",
	}

	pods, err := loadPods(ctx, key, o, selector)
	if err != nil {
		return nil, errors.Wrap(err, "load pods")
	}

	var owned []*corev1.Pod
	for _, pod := range pods {
		controllerRef := metav1.GetControllerOf(pod)
		if controllerRef == nil || controllerRef.UID != uid {
			continue
		}

		owned = append(owned, pod)
	}

	return owned, nil
}

func loadPods(ctx context.Context, key store.Key, o store.Store, labelSelector *metav1.LabelSelector) ([]*corev1.Pod, error) {
	objects, _, err := o.List(ctx, key)
	if err != nil {
		return nil, err
	}

	var list []*corev1.Pod

	for i := range objects.Items {
		pod := &corev1.Pod{}

		if err := kubernetes.FromUnstructured(&objects.Items[i], pod); err != nil {
			return nil, err
		}

		podSelector := &metav1.LabelSelector{
			MatchLabels: pod.GetLabels(),
		}

		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, err
		}

		if selector == kLabels.Nothing() || isEqualSelector(labelSelector, podSelector) || selector.Matches(kLabels.Set(pod.Labels)) {
			list = append(list, pod)
		}
	}

	return list, nil
}

// extraKeys are keys that should be ignored in labels. These keys are added
// by tools or by Kubernetes itself.
var extraKeys = []string{
	"statefulset.kubernetes.io/pod-name",
	appsv1.DefaultDeploymentUniqueLabelKey,
	"controller-revision-hash",
	"pod-template-generation",
}

func isEqualSelector(s1, s2 *metav1.LabelSelector) bool {
	if s1 == nil || s2 == nil {
		return false
	}

	s1Copy := s1.DeepCopy()
	s2Copy := s2.DeepCopy()

	for _, key := range extraKeys {
		delete(s1Copy.MatchLabels, key)
		delete(s2Copy.MatchLabels, key)
	}

	return apiEquality.Semantic.DeepEqual(s1Copy, s2Copy)
}

func createPodListView(ctx context.Context, object runtime.Object, options Options) (component.Component, error) {
	options.DisableLabels = true

	podList := &corev1.PodList{}

	objectStore := options.DashConfig.ObjectStore()

	accessor := meta.NewAccessor()

	namespace, err := accessor.Namespace(object)
	if err != nil {
		return nil, errors.Wrap(err, "get namespace for object")
	}

	apiVersion, err := accessor.APIVersion(object)
	if err != nil {
		return nil, errors.Wrap(err, "Get apiVersion for object")
	}

	kind, err := accessor.Kind(object)
	if err != nil {
		return nil, errors.Wrap(err, "get kind for object")
	}

	name, err := accessor.Name(object)
	if err != nil {
		return nil, errors.Wrap(err, "get name for object")
	}

	key := store.Key{
		Namespace:  namespace,
		APIVersion: "v1",
		Kind:       "Pod",
	}

	list, _, err := objectStore.List(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "list all objects for key %+v", key)
	}

	for i := range list.Items {
		pod := &corev1.Pod{}
		err := kubernetes.FromUnstructured(&list.Items[i], pod)
		if err != nil {
			return nil, err
		}

		for _, ownerReference := range pod.OwnerReferences {
			if ownerReference.APIVersion == apiVersion &&
				ownerReference.Kind == kind &&
				ownerReference.Name == name {
				podList.Items = append(podList.Items, *pod)
			}
		}
	}

	return PodListHandler(ctx, podList, options)
}

func createRollingPodListView(ctx context.Context, objects []runtime.Object, options Options) (component.Component, error) {
	options.DisableLabels = true

	podList := &corev1.PodList{}

	objectStore := options.DashConfig.ObjectStore()

	accessor := meta.NewAccessor()

	for _, object := range objects {
		namespace, err := accessor.Namespace(object)
		if err != nil {
			return nil, errors.Wrap(err, "get namespace for object")
		}

		apiVersion, err := accessor.APIVersion(object)
		if err != nil {
			return nil, errors.Wrap(err, "Get apiVersion for object")
		}

		kind, err := accessor.Kind(object)
		if err != nil {
			return nil, errors.Wrap(err, "get kind for object")
		}

		name, err := accessor.Name(object)
		if err != nil {
			return nil, errors.Wrap(err, "get name for object")
		}

		key := store.Key{
			Namespace:  namespace,
			APIVersion: "v1",
			Kind:       "Pod",
		}

		list, _, err := objectStore.List(ctx, key)
		if err != nil {
			return nil, errors.Wrapf(err, "list all objects for key %+v", key)
		}

		for i := range list.Items {
			pod := &corev1.Pod{}
			err := kubernetes.FromUnstructured(&list.Items[i], pod)
			if err != nil {
				return nil, err
			}

			for _, ownerReference := range pod.OwnerReferences {
				if ownerReference.APIVersion == apiVersion &&
					ownerReference.Kind == kind &&
					ownerReference.Name == name {
					podList.Items = append(podList.Items, *pod)
				}
			}
		}
	}

	return PodListHandler(ctx, podList, options)
}

func createMountedPodListView(ctx context.Context, namespace string, persistentVolumeClaimName string, options Options) (component.Component, error) {
	options.DisableLabels = true

	key := store.Key{
		Namespace:  namespace,
		APIVersion: "v1",
		Kind:       "Pod",
	}

	objectStore := options.DashConfig.ObjectStore()

	mountedPodList := &corev1.PodList{}

	pods, err := loadPods(ctx, key, objectStore, nil)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		var volumeClaims []corev1.Volume

		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil {
				volumeClaims = append(volumeClaims, volume)
			}
		}

		for _, persistentVolumeClaim := range volumeClaims {
			if persistentVolumeClaim.PersistentVolumeClaim.ClaimName == persistentVolumeClaimName {
				mountedPodList.Items = append(mountedPodList.Items, *pod)
			}
		}
	}

	return PodListHandler(ctx, mountedPodList, options)
}

func printPodResources(podSpec corev1.PodSpec) (*component.Table, error) {
	table := component.NewTable("Resources", "Pod has no resource needs", podResourceCols)

	// for each container in the spec, there will be requests and limits
	// for memory and cpu

	for _, container := range podSpec.Containers {
		memoryRequest := ""
		if q := container.Resources.Requests.Memory(); q != nil {
			memoryRequest = q.String()
		}
		cpuRequest := ""
		if q := container.Resources.Requests.Cpu(); q != nil {
			cpuRequest = q.String()
		}
		memoryLimit := ""
		if q := container.Resources.Limits.Memory(); q != nil {
			memoryLimit = q.String()
		}
		cpuLimit := ""
		if q := container.Resources.Limits.Cpu(); q != nil {
			cpuLimit = q.String()
		}

		row := component.TableRow{
			"Container":       component.NewText(container.Name),
			"Request: Memory": component.NewText(memoryRequest),
			"Request: CPU":    component.NewText(cpuRequest),
			"Limit: Memory":   component.NewText(memoryLimit),
			"Limit: CPU":      component.NewText(cpuLimit),
		}
		table.Add(row)
	}

	return table, nil
}

type podObject interface {
	Config(options Options) error
	Status(options Options) error
	InitContainers(ctx context.Context, options Options) error
	Containers(ctx context.Context, options Options) error
	Additional(options Options) error
}

type podHandler struct {
	pod             *corev1.Pod
	configFunc      func(*corev1.Pod, Options) (*component.Summary, error)
	summaryFunc     func(*corev1.Pod, Options) (*component.Summary, error)
	containerFunc   func(ctx context.Context, pod *corev1.Pod, container *corev1.Container, isInit bool, options Options) (*component.Summary, error)
	additionalFuncs []func(*corev1.Pod, Options) ObjectPrinterFunc
	object          *Object
}

var _ podObject = (*podHandler)(nil)

var defaultPodHandlerAdditionalItems = []func(*corev1.Pod, Options) ObjectPrinterFunc{
	func(pod *corev1.Pod, options Options) ObjectPrinterFunc {
		return func() (component.Component, error) {
			return printPodResources(pod.Spec)
		}
	},
	func(pod *corev1.Pod, options Options) ObjectPrinterFunc {
		return func() (component.Component, error) {
			return printVolumes(pod.Spec.Volumes)
		}
	},
	func(pod *corev1.Pod, options Options) ObjectPrinterFunc {
		return func() (component.Component, error) {
			return printTolerations(pod.Spec)
		}
	},
	func(pod *corev1.Pod, options Options) ObjectPrinterFunc {
		return func() (component.Component, error) {
			return printAffinity(pod.Spec)
		}
	},
}

func newPodHandler(pod *corev1.Pod, object *Object) (*podHandler, error) {
	if pod == nil {
		return nil, errors.New("can't print a nil pod")
	}

	if object == nil {
		return nil, errors.New("can't print pod using a nil object printer")
	}

	ph := &podHandler{
		pod:             pod,
		configFunc:      defaultPodConfig,
		summaryFunc:     defaultPodSummary,
		containerFunc:   defaultPodContainers,
		additionalFuncs: defaultPodHandlerAdditionalItems,
		object:          object,
	}

	return ph, nil
}

func (p *podHandler) Config(options Options) error {
	out, err := p.configFunc(p.pod, options)
	if err != nil {
		return err
	}
	p.object.RegisterConfig(out)
	return nil
}

func defaultPodConfig(pod *corev1.Pod, options Options) (*component.Summary, error) {
	creator := NewPodConfiguration(pod)
	return creator.Create(options)
}

func (p *podHandler) Status(options Options) error {
	out, err := p.summaryFunc(p.pod, options)
	if err != nil {
		return err
	}

	p.object.RegisterSummary(out)
	return nil
}

func defaultPodSummary(pod *corev1.Pod, options Options) (*component.Summary, error) {
	return createPodSummaryStatus(pod)
}

func (p *podHandler) InitContainers(ctx context.Context, options Options) error {
	return p.containers(ctx, p.pod.Spec.InitContainers, true, options)
}

func (p *podHandler) EphemeralContainers(ctx context.Context, options Options) error {
	var containers []corev1.Container
	for _, container := range p.pod.Spec.EphemeralContainers {
		containers = append(containers, corev1.Container(container.EphemeralContainerCommon))
	}
	return p.containers(ctx, containers, false, options)
}

func (p *podHandler) containers(ctx context.Context, containers []corev1.Container, isInit bool, options Options) error {
	var itemDescriptors []ItemDescriptor

	for i := range containers {
		container := containers[i]

		itemDescriptors = append(itemDescriptors, ItemDescriptor{
			Width: component.WidthFull,
			Func: func() (component.Component, error) {
				return p.containerFunc(ctx, p.pod, &container, isInit, options)
			},
		})
	}

	p.object.RegisterItems(itemDescriptors...)

	return nil
}

func (p *podHandler) Containers(ctx context.Context, options Options) error {
	return p.containers(ctx, p.pod.Spec.Containers, false, options)
}

func defaultPodContainers(ctx context.Context, pod *corev1.Pod, container *corev1.Container, isInit bool, options Options) (*component.Summary, error) {
	portForwarder := options.DashConfig.PortForwarder()
	creator := NewContainerConfiguration(ctx, pod, container, portForwarder, IsInit(isInit), WithPrintOptions(options))
	return creator.Create()
}

func (p *podHandler) Additional(options Options) error {
	var itemDescriptors []ItemDescriptor

	for i := range p.additionalFuncs {
		itemDescriptors = append(itemDescriptors, ItemDescriptor{
			Width: component.WidthHalf,
			Func:  p.additionalFuncs[i](p.pod, options),
		})
	}

	p.object.RegisterItems(itemDescriptors...)

	return nil
}

// Metrics adds CPU and Memory Summary cards when metrics-server is available.
func (p *podHandler) Metrics(ctx context.Context, options Options) error {
	pml, err := octant.NewClusterPodMetricsLoader(options.DashConfig.ClusterClient())
	if err != nil {
		log.From(ctx).Warnf("create pod metrics loader: %v", err)
		return nil
	}

	supported, err := pml.SupportsMetrics(ctx)
	if err != nil || !supported {
		return nil
	}

	metricsObj, found, err := pml.Load(ctx, p.pod.Namespace, p.pod.Name)
	if err != nil {
		log.From(ctx).Warnf("load pod metrics for %s/%s: %v", p.pod.Namespace, p.pod.Name, err)
		return nil
	}
	if !found {
		return nil
	}

	containersRaw, found, err := unstructured.NestedSlice(metricsObj.Object, "containers")
	if err != nil || !found {
		return nil
	}

	cpuUsage := resource.Quantity{}
	memUsage := resource.Quantity{}

	for _, c := range containersRaw {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		usage, found, err := unstructured.NestedMap(container, "usage")
		if err != nil || !found {
			continue
		}
		if q, err := podMetricQuantity(usage, "cpu"); err == nil {
			cpuUsage.Add(q)
		}
		if q, err := podMetricQuantity(usage, "memory"); err == nil {
			memUsage.Add(q)
		}
	}

	// Sum requests/limits across containers
	var cpuReq, cpuLim, memReq, memLim resource.Quantity
	for _, c := range p.pod.Spec.Containers {
		if q := c.Resources.Requests.Cpu(); q != nil {
			cpuReq.Add(*q)
		}
		if q := c.Resources.Limits.Cpu(); q != nil {
			cpuLim.Add(*q)
		}
		if q := c.Resources.Requests.Memory(); q != nil {
			memReq.Add(*q)
		}
		if q := c.Resources.Limits.Memory(); q != nil {
			memLim.Add(*q)
		}
	}

	cpuStatus := podMetricStatus(cpuUsage.MilliValue(), cpuReq.MilliValue(), cpuLim.MilliValue())
	memStatus := podMetricStatus(memUsage.Value(), memReq.Value(), memLim.Value())

	cpuSummary := component.NewSummary("CPU")
	cpuSections := component.SummarySections{}
	cpuSections.Add("Current", podMetricValueText(fmt.Sprintf("%vm", cpuUsage.MilliValue()), cpuStatus))
	if !cpuReq.IsZero() {
		cpuSections.AddText("Request", fmt.Sprintf("%vm", cpuReq.MilliValue()))
	}
	if !cpuLim.IsZero() {
		cpuSections.AddText("Limit", fmt.Sprintf("%vm", cpuLim.MilliValue()))
	}
	cpuSummary.Add(cpuSections...)

	memSummary := component.NewSummary("Memory")
	memSections := component.SummarySections{}
	memSections.Add("Current", podMetricValueText(fmt.Sprintf("%vMi", memUsage.Value()/(1024*1024)), memStatus))
	if !memReq.IsZero() {
		memSections.AddText("Request", fmt.Sprintf("%vMi", memReq.Value()/(1024*1024)))
	}
	if !memLim.IsZero() {
		memSections.AddText("Limit", fmt.Sprintf("%vMi", memLim.Value()/(1024*1024)))
	}
	memSummary.Add(memSections...)

	p.object.RegisterItems(
		ItemDescriptor{Width: component.WidthHalf, Component: cpuSummary},
		ItemDescriptor{Width: component.WidthHalf, Component: memSummary},
	)

	return nil
}

// podMetricQuantity extracts and parses a resource.Quantity from a metrics usage map.
func podMetricQuantity(usage map[string]interface{}, field string) (resource.Quantity, error) {
	s, found, err := unstructured.NestedString(usage, field)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("parse %s: %w", field, err)
	}
	if !found {
		return resource.Quantity{}, fmt.Errorf("field %s not found", field)
	}
	return resource.ParseQuantity(s)
}

// podMetricStatus computes the display status for a resource metric value given
// its request and limit (all in the same unit — millivalue for CPU, value for memory).
//
// Rules:
//   - Both unset → 0 (black)
//   - Request set, within 5% of limit → Error (red, takes precedence over yellow)
//   - Request unset, limit set, within 10% of limit → Error (red)
//   - Current < request → OK (green)
//   - Current ≥ request → Warning (yellow), whether or not limit is set
func podMetricStatus(current, request, limit int64) component.TextStatus {
	hasReq := request > 0
	hasLim := limit > 0

	if !hasReq && !hasLim {
		return 0 // black
	}

	// Red: request set and within 5% of limit (takes precedence over yellow)
	if hasReq && hasLim && current*100 >= limit*95 {
		return component.TextStatusError
	}

	// Red: no request but has limit, within 10%
	if !hasReq && hasLim && current*100 >= limit*90 {
		return component.TextStatusError
	}

	if !hasReq {
		// Has limit but not in red zone; no request to compare against
		return 0 // black
	}

	if current < request {
		return component.TextStatusOK // green
	}

	return component.TextStatusWarning // yellow
}

// podMetricValueText creates a Text component with a CSS class for status color.
// No markdown or HTML is used so the value stays column-aligned with plain rows.
func podMetricValueText(value string, status component.TextStatus) *component.Text {
	classes := map[component.TextStatus]string{
		component.TextStatusOK:      "text-metric-ok",
		component.TextStatusWarning: "text-metric-warning",
		component.TextStatusError:   "text-metric-error",
	}
	t := component.NewText(value)
	if class, ok := classes[status]; ok {
		t.SetClassName(class)
	}
	return t
}

func podTableFilters() map[string]component.TableFilter {
	return map[string]component.TableFilter{
		"Status": {
			Values:   []string{"Pending", "Running", "Succeeded", "Failed", "Unknown"},
			Selected: []string{"Pending", "Running"},
		},
	}
}

func addPodTableFilters(table *component.Table) {
	for k, v := range podTableFilters() {
		table.AddFilter(k, v)
	}
}
