package app

import (
	"context"
	unicore "github.com/mcyouyou/unicore/api/deployer/v1"
	apps "k8s.io/api/apps/v1"
)

func (c *StateController) doImagePreLoad(ctx context.Context,
	app *unicore.App,
	currentRevision *apps.ControllerRevision,
	updateRevision *apps.ControllerRevision) {
	if currentRevision.Name != updateRevision.Name {
		// TODO
	}
}
