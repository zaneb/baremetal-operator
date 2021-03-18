package coreos

import (
	"github.com/go-logr/logr"

	"github.com/metal3-io/baremetal-operator/pkg/provisioner"
)

type CoreOSProvisionerFactory struct {
	DelegateFactory provisioner.Factory
	ServiceURL      string
}

type isoURLFetcher func() (string, error)

// New returns a new Empty Provisioner
func (f CoreOSProvisionerFactory) New(hostData provisioner.HostData, publisher provisioner.EventPublisher) (provisioner.Provisioner, error) {
	delegate, err := f.DelegateFactory(hostData, publisher)
	logger := log.WithValues("host", hostData.ObjectMeta.Name)
	return &coreosProvisioner{
		Provisioner: delegate,
		objectMeta:  hostData.ObjectMeta,
		provID:      hostData.ProvisionerID,
		getISOURL:   f.MakeISOURLFetcher(hostData, logger),
		log:         logger,
		publisher:   publisher,
	}, err
}

func (f CoreOSProvisionerFactory) MakeISOURLFetcher(hostData provisioner.HostData, log logr.Logger) isoURLFetcher {
	return func() (string, error) {
		// TODO: Get a URL specific to this image from the assisted service
		return f.ServiceURL, nil
	}
}
