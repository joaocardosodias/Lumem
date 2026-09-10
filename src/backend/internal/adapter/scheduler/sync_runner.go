package scheduler

import (
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.JobRunner = (*SyncJobRunner)(nil)

type SyncJobRunner struct{}

func NewSyncJobRunner() *SyncJobRunner {
	return &SyncJobRunner{}
}

func (r *SyncJobRunner) Run(job func()) {
	job()
}
