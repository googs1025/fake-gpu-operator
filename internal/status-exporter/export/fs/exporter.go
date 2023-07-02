package fs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/run-ai/fake-gpu-operator/internal/common/constants"
	"github.com/run-ai/fake-gpu-operator/internal/common/topology"
	"github.com/run-ai/fake-gpu-operator/internal/status-exporter/export"
	"github.com/run-ai/fake-gpu-operator/internal/status-exporter/watch"
)

type FsExporter struct {
	topologyChan <-chan *topology.NodeTopology
}

var _ export.Interface = &FsExporter{}

func NewFsExporter(watcher watch.Interface) *FsExporter {
	topologyChan := make(chan *topology.NodeTopology)
	watcher.Subscribe(topologyChan)

	return &FsExporter{
		topologyChan: topologyChan,
	}
}

func (e *FsExporter) Run(stopCh <-chan struct{}) {
	for {
		select {
		case nodeTopology := <-e.topologyChan:
			e.export(nodeTopology)
		case <-stopCh:
			return
		}
	}
}

func (e *FsExporter) export(nodeTopology *topology.NodeTopology) {

	for gpuIdx, gpu := range nodeTopology.Gpus {
		// Ignoring pods that are not supposed to be seen by runai-container-toolkit
		if gpu.Status.AllocatedBy.Namespace != constants.ReservationNs {
			continue
		}

		for podUuid, gpuUsageStatus := range gpu.Status.PodGpuUsageStatus {
			log.Printf("Exporting pod %s gpu utilization to filesystem", podUuid)
			utilization := gpuUsageStatus.Utilization.Random()

			path := fmt.Sprintf("runai/proc/pod/%s/metrics/gpu/%d/utilization.sm", podUuid, gpuIdx)
			if err := os.MkdirAll(filepath.Dir(path), 0644); err != nil {
				log.Printf("Failed creating directory for pod %s: %s", podUuid, err.Error())
			}

			if err := os.WriteFile(path, []byte(strconv.Itoa(utilization)), 0644); err != nil {
				log.Printf("Failed exporting pod %s to filesystem: %s", podUuid, err.Error())
			}
		}
	}
}
