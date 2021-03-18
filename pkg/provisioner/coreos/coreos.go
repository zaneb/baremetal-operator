package coreos

import (
	"errors"
	"time"

	"github.com/go-logr/logr"
	logz "sigs.k8s.io/controller-runtime/pkg/log/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/metal3-io/baremetal-operator/pkg/hardware"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner"
)

const delay = time.Second * 10

var log = logz.New().WithName("provisioner").WithName("coreos")

type coreosProvisioner struct {
	// the delegate provisioner
	provisioner.Provisioner
	// the object metadata of the BareMetalHost resource
	objectMeta metav1.ObjectMeta
	// the ID of the host in the delegate provisioner
	provID string
	// function to get the URL of the ISO image containing the agent
	getISOURL isoURLFetcher
	// a logger configured for this host
	log logr.Logger
	// an event publisher for recording significant events
	publisher provisioner.EventPublisher
}

func (p *coreosProvisioner) findAgent() (agent, error) {
	// TODO: Look up the agent and return something
	return nil, nil
}

func (p *coreosProvisioner) delegateState(hostState metal3v1alpha1.ProvisioningState) (state metal3v1alpha1.ProvisioningState, err error) {
	state = hostState
	switch hostState {
	case metal3v1alpha1.StateRegistering:
		state = metal3v1alpha1.StateProvisioning
	case metal3v1alpha1.StateDeprovisioning:
		var agent agent
		agent, err = p.findAgent()
		if agent == nil {
			state = metal3v1alpha1.StateProvisioning
		}
	default:
		state = metal3v1alpha1.StateProvisioned
	}
	return
}

func (p *coreosProvisioner) delegateProvisionData() (data provisioner.ProvisionData, err error) {
	diskFormat := "live-iso"
	hwProfile, err := hardware.GetProfile("libvirt")
	if err != nil {
		return
	}
	isoURL, err := p.getISOURL()
	if err != nil {
		return
	}
	data = provisioner.ProvisionData{
		Image: metal3v1alpha1.Image{
			URL:        isoURL,
			DiskFormat: &diskFormat,
		},
		BootMode:        metal3v1alpha1.UEFI,
		HardwareProfile: hwProfile, // TODO: replace with proper root device hints
	}
	return
}

func (p *coreosProvisioner) bootAgent() (result provisioner.Result, err error) {
	provData, err := p.delegateProvisionData()
	result, err = p.Provisioner.Provision(provData)
	if isIncomplete(result, err) {
		return
	}

	agent, err := p.findAgent()
	if err != nil {
		return
	}
	if agent == nil {
		result = provisioner.Result{Dirty: true, RequeueAfter: delay}
	}
	return
}

// ValidateManagementAccess tests the connection information for the
// host to verify that the location and credentials work.
func (p *coreosProvisioner) ValidateManagementAccess(data provisioner.ManagementAccessData, credentialsChanged, force bool) (result provisioner.Result, id string, err error) {
	isInitialising := data.State == metal3v1alpha1.StateRegistering
	data.State, err = p.delegateState(data.State)
	if err != nil {
		return
	}

	result, id, err = p.Provisioner.ValidateManagementAccess(data, credentialsChanged, force)
	if isIncomplete(result, err) || id != p.provID || !isInitialising {
		return
	}

	result, err = p.bootAgent()
	return
}

// Adopt notifies the provisioner that the state machine believes the host
// to be currently provisioned, and that it should be managed as such.
func (p *coreosProvisioner) Adopt(data provisioner.AdoptData, force bool) (result provisioner.Result, err error) {
	data.State, err = p.delegateState(data.State)
	if err != nil {
		return
	}
	return p.Provisioner.Adopt(data, force)
}

// InspectHardware updates the HardwareDetails field of the host with
// details of devices discovered on the hardware. It may be called
// multiple times, and should return true for its dirty flag until the
// inspection is completed.
func (p *coreosProvisioner) InspectHardware(data provisioner.InspectData, force bool) (result provisioner.Result, details *metal3v1alpha1.HardwareDetails, err error) {
	agent, err := p.findAgent()
	if err != nil {
		return
	}
	if agent == nil {
		result = provisioner.Result{Dirty: true, RequeueAfter: delay}
		return
	}
	details, err = agent.HardwareDetails()
	return
}

// Prepare remove existing configuration and set new configuration
func (p *coreosProvisioner) Prepare(data provisioner.PrepareData, unprepared bool) (result provisioner.Result, started bool, err error) {
	if data.RAIDConfig != nil {
		err = errors.New("RAID settings are defined, but RAID is not supported")
	}
	return
}

// Provision writes the image from the host spec to the host. It may
// be called multiple times, and should return true for its dirty flag
// until the deprovisioning operation is completed.
func (p *coreosProvisioner) Provision(data provisioner.ProvisionData) (result provisioner.Result, err error) {
	// TODO: force power on?
	agent, err := p.findAgent()
	if err != nil {
		return
	}
	complete, err := agent.Deploy(data.Image.URL)
	result = provisioner.Result{Dirty: !complete, RequeueAfter: delay}
	return
}

// Deprovision removes the host from the image. It may be called
// multiple times, and should return true for its dirty flag until the
// deprovisioning operation is completed.
func (p *coreosProvisioner) Deprovision(force bool) (result provisioner.Result, err error) {
	agent, err := p.findAgent()
	if err != nil {
		return
	}

	deployed := false
	if agent != nil {
		deployed, err = agent.IsDeployed()
		if err != nil {
			return
		}
	}
	if deployed {
		result, err = p.Provisioner.Deprovision(force)
		if err != nil || result.Dirty {
			return
		}
		agent.CleanUp()
	}
	return p.bootAgent()
}

func isIncomplete(result provisioner.Result, err error) bool {
	return err != nil ||
		result.Dirty ||
		result.ErrorMessage != ""
}
