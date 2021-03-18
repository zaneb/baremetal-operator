package coreos

import (
	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
)

// TODO: Implement a real agent type
type agent interface {
	HardwareDetails() (*metal3v1alpha1.HardwareDetails, error)
	IsDeployed() (bool, error)
	Deploy(url string) (bool, error)
	CleanUp() error
}
