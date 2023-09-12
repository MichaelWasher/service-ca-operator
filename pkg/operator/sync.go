package operator

import (
	"context"

	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"k8s.io/klog/v2"
)

func (c *serviceCAOperator) syncControllers(ctx context.Context, operatorConfig *operatorv1.ServiceCA, infrastructure *configv1.Infrastructure) error {
	// Any modification of resource we want to trickle down to force deploy all of the controllers.
	// Sync the controller NS and the other resources. These should be mostly static.
	needsDeploy, err := c.manageControllerNS(ctx)
	if err != nil {
		return err
	}

	err = c.manageControllerResources(ctx, &needsDeploy)
	if err != nil {
		return err
	}

	// Sync the CA (regenerate if missing).
	caModified, err := c.manageSignerCA(ctx, operatorConfig.Spec.UnsupportedConfigOverrides.Raw)
	if err != nil {
		return err
	}
	klog.V(4).Infof("Starting sleep")
	for i := 0; i < 6; i++ {
		klog.V(4).Infof("Sleep number: %q", i)
		time.Sleep(10 * time.Second)
	}

	klog.V(4).Infof("Ending sleep")

	// Sync the CA bundle. This will be updated if the CA has changed.
	_, err = c.manageSignerCABundle(ctx, caModified)
	if err != nil {
		return err
	}

	// Sync the controller.
	_, err = c.manageDeployment(ctx, operatorConfig, needsDeploy || caModified, shouldScheduleOnWorkers(infrastructure))
	if err != nil {
		return err
	}

	klog.V(4).Infof("synced all controller resources")
	return nil
}

func shouldScheduleOnWorkers(infra *configv1.Infrastructure) bool {
	return infra.Status.ControlPlaneTopology == configv1.ExternalTopologyMode
}
