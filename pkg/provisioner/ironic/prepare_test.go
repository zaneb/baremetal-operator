package ironic

import (
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
	"github.com/gophercloud/gophercloud/openstack/baremetalintrospection/v1/introspection"
	"github.com/stretchr/testify/assert"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/metal3-io/baremetal-operator/pkg/bmc"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/clients"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/testserver"
)

func TestPrepare(t *testing.T) {
	nodeUUID := "33ce8659-7400-4c68-9535-d10766f07a58"
	cases := []struct {
		name                 string
		ironic               *testserver.IronicMock
		unprepared           bool
		existRaidConfig      bool
		expectedStarted      bool
		expectedDirty        bool
		expectedError        bool
		expectedRequestAfter int
	}{
		{
			name: "manageable state(haven't clean steps)",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.Manageable),
				UUID:           nodeUUID,
			}),
			unprepared:           true,
			expectedStarted:      false,
			expectedRequestAfter: 0,
			expectedDirty:        false,
		},
		{
			name: "manageable state(have clean steps)",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.Manageable),
				UUID:           nodeUUID,
			}),
			unprepared:           true,
			existRaidConfig:      true,
			expectedStarted:      true,
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "cleanFail state(cleaned provision settings)",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.CleanFail),
				UUID:           nodeUUID,
			}),
			expectedStarted:      false,
			expectedRequestAfter: 0,
			expectedDirty:        false,
		},
		{
			name: "cleanFail state(set ironic host to manageable)",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.CleanFail),
				UUID:           nodeUUID,
			}),
			unprepared:           true,
			existRaidConfig:      true,
			expectedStarted:      false,
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "cleaning state",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.Cleaning),
				UUID:           nodeUUID,
			}),
			existRaidConfig:      true,
			expectedStarted:      false,
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "cleanWait state",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.CleanWait),
				UUID:           nodeUUID,
			}),
			existRaidConfig:      true,
			expectedStarted:      false,
			expectedRequestAfter: 10,
			expectedDirty:        true,
		},
		{
			name: "manageable state(manual clean finished)",
			ironic: testserver.NewIronic(t).WithDefaultResponses().Node(nodes.Node{
				ProvisionState: string(nodes.Manageable),
				UUID:           nodeUUID,
			}),
			existRaidConfig:      true,
			expectedStarted:      false,
			expectedRequestAfter: 0,
			expectedDirty:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ironic != nil {
				tc.ironic.Start()
				defer tc.ironic.Stop()
			}

			inspector := testserver.NewInspector(t).Ready().WithIntrospection(nodeUUID, introspection.Introspection{
				Finished: false,
			})
			inspector.Start()
			defer inspector.Stop()

			host := makeHost()
			host.Status.Provisioning.ID = nodeUUID
			prepData := provisioner.PrepareData{}
			if tc.existRaidConfig {
				host.Spec.BMC.Address = "irmc://test.bmc/"
				prepData.RAIDConfig = &metal3v1alpha1.RAIDConfig{
					HardwareRAIDVolumes: []metal3v1alpha1.HardwareRAIDVolume{
						{
							Name:  "root",
							Level: "1",
						},
						{
							Name:  "v1",
							Level: "1",
						},
					},
				}
			}

			auth := clients.AuthConfig{Type: clients.NoAuth}
			prov, err := newProvisionerWithSettings(
				provisioner.BuildHostData(host, bmc.Credentials{}), nullEventPublisher,
				tc.ironic.Endpoint(), auth, inspector.Endpoint(), auth,
			)
			if err != nil {
				t.Fatalf("could not create provisioner: %s", err)
			}

			result, started, err := prov.Prepare(prepData, tc.unprepared)

			assert.Equal(t, tc.expectedStarted, started)
			assert.Equal(t, tc.expectedDirty, result.Dirty)
			assert.Equal(t, time.Second*time.Duration(tc.expectedRequestAfter), result.RequeueAfter)
			if !tc.expectedError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
