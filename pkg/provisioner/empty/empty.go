package empty

import (
	logz "sigs.k8s.io/controller-runtime/pkg/log/zap"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner"
)

var log = logz.New().WithName("provisioner").WithName("empty")

// Provisioner implements the provisioning.Provisioner interface
type emptyProvisioner struct {
}

// New returns a new Empty Provisioner
func New(hostData provisioner.HostData, publisher provisioner.EventPublisher) (provisioner.Provisioner, error) {
	return &emptyProvisioner{}, nil
}

// ValidateManagementAccess tests the connection information for the
// host to verify that the location and credentials work.
func (p *emptyProvisioner) ValidateManagementAccess(data provisioner.ManagementAccessData, credentialsChanged, force bool) (provisioner.Result, string, error) {
	return provisioner.Result{}, "", nil
}

// InspectHardware updates the HardwareDetails field of the host with
// details of devices discovered on the hardware. It may be called
// multiple times, and should return true for its dirty flag until the
// inspection is completed.
func (p *emptyProvisioner) InspectHardware(data provisioner.InspectData, force bool) (provisioner.Result, *metal3v1alpha1.HardwareDetails, error) {
	return provisioner.Result{}, nil, nil
}

// UpdateHardwareState fetches the latest hardware state of the server
// and updates the HardwareDetails field of the host with details. It
// is expected to do this in the least expensive way possible, such as
// reading from a cache.
func (p *emptyProvisioner) UpdateHardwareState() (provisioner.HardwareState, error) {
	return provisioner.HardwareState{}, nil
}

// Adopt notifies the provisioner that the state machine believes the host
// to be currently provisioned, and that it should be managed as such.
func (p *emptyProvisioner) Adopt(data provisioner.AdoptData, force bool) (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// Prepare remove existing configuration and set new configuration
func (p *emptyProvisioner) Prepare(data provisioner.PrepareData, unprepared bool) (result provisioner.Result, started bool, err error) {
	return provisioner.Result{}, false, nil
}

// Provision writes the image from the host spec to the host. It may
// be called multiple times, and should return true for its dirty flag
// until the deprovisioning operation is completed.
func (p *emptyProvisioner) Provision(data provisioner.ProvisionData) (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// Deprovision removes the host from the image. It may be called
// multiple times, and should return true for its dirty flag until the
// deprovisioning operation is completed.
func (p *emptyProvisioner) Deprovision(force bool) (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// Delete removes the host from the provisioning system. It may be
// called multiple times, and should return true for its dirty flag
// until the deprovisioning operation is completed.
func (p *emptyProvisioner) Delete() (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// PowerOn ensures the server is powered on independently of any image
// provisioning operation.
func (p *emptyProvisioner) PowerOn() (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// PowerOff ensures the server is powered off independently of any image
// provisioning operation.
func (p *emptyProvisioner) PowerOff(rebootMode metal3v1alpha1.RebootMode) (provisioner.Result, error) {
	return provisioner.Result{}, nil
}

// IsReady always returns true for the empty provisioner
func (p *emptyProvisioner) IsReady() (bool, error) {
	return true, nil
}

func (p *emptyProvisioner) HasProvisioningCapacity() (result bool, err error) {
	return true, nil
}
