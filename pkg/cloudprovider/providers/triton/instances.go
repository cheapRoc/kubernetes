package triton

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/golang/glog"
	triton "github.com/joyent/triton-go"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/types"
)

type Instances struct {
	provider *Triton
}

//
// -----------------------------------------------------------------------------
//

// getMachineByUUID returns the triton.Machine for a given UUID
func (i Instances) getMachineByUUID(uuid string) (*triton.Machine, error) {
	input := &triton.GetMachineInput{uuid}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, err
	}
	return machine, nil
}

// getMachineByName searches through all machines within a Triton account
// looking for a match by passed in serverName string. Checks (within order)
// PrimaryIP, Hostname, and UUID.
func (i Instances) getMachineByName(serverName string) (*triton.Machine, error) {
	input := &triton.ListMachinesInput{}
	machines, err := i.provider.Client.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.Errorf("Machines.ListMachines() returned an error: %s", err)
		return nil, err
	}

	for _, machine := range machines {
		if machine.PrimaryIP == serverName {
			return machine, nil
		}
		if machine.Name == serverName {
			return machine, nil
		}
		// TODO: make this a short UUID match
		if machine.ID == serverName {
			return machine, nil
		}
	}

	return nil, fmt.Errorf("No machine found by serverName: %s", serverName)
}

// getMachineByHostname searches through all machines within a Triton account
// looking for a match by passed in hostname string. Checks only hostname.
func (i Instances) getMachineByHostname(hostname string) (*triton.Machine, error) {
	input := &triton.ListMachinesInput{}
	machines, err := i.provider.Client.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.Errorf("Machines.ListMachines() returned an error: %s", err)
		return nil, err
	}

	for _, machine := range machines {
		if machine.Name == hostname {
			return machine, nil
		}
		// TODO: make this a short UUID match
		if machine.ID == hostname {
			return machine, nil
		}
	}
	return nil, fmt.Errorf("No machine found by hostname: %s", hostname)
}

// probeNodeAddress returns the IP address for the current metadata UUID or a
// given serverName.
func (i Instances) probeNodeAddress(serverName string) (string, error) {
	if i.provider.Metadata.Hostname == serverName {
		machine, err := i.getMachineByUUID(i.provider.Metadata.UUID)
		if err != nil {
			return "", err
		}
		if machine.State != "running" {
			return "", cloudprovider.InstanceNotFound
		}
		return machine.PrimaryIP, nil
	}

	machine, err := i.getMachineByName(serverName)
	if err != nil {
		return "", err
	}
	return machine.PrimaryIP, nil
}

// probeMachineUUID probes the existing machine's metadata for a UUID and
// returns, otherwise we search through all machines and return an ID that way.
func (i Instances) probeMachineUUID(serverName string) (string, error) {
	if i.provider.Metadata.Hostname == serverName {
		machine, err := i.getMachineByUUID(i.provider.Metadata.UUID)
		if err != nil {
			return "", err
		}
		if machine.State != "running" {
			return "", cloudprovider.InstanceNotFound
		}
		return machine.ID, nil
	}

	machine, err := i.getMachineByName(serverName)
	if err != nil {
		return "", cloudprovider.InstanceNotFound
	}
	return machine.ID, nil
}

// probeMachineBrand returns the Triton brand of the machine for the current
// metadata UUID or given serverName.
func (i Instances) probeMachineBrand(serverName string) (string, error) {
	if i.provider.Metadata.Hostname == serverName {
		machine, err := i.getMachineByUUID(i.provider.Metadata.UUID)
		if err != nil {
			return "", err
		}
		if machine.State != "running" {
			return "", cloudprovider.InstanceNotFound
		}
		return machine.Brand, nil
	}

	machine, err := i.getMachineByName(serverName)
	if err != nil {
		return "", err
	}
	return machine.Brand, nil
}

//
// -----------------------------------------------------------------------------
//

// Instances returns all known cloud providers instances running for our
// configured Triton client.
func (t *Triton) Instances() (cloudprovider.Instances, bool) {
	glog.V(2).Info("Triton.Instances() called")

	return &Instances{
		provider: t,
	}, true
}

// NodeAddresses returns the addresses of the specified instance.
//
// NOTE: This currently is only used in such a way that it returns the address
// of the calling instance. We should do a rename to make this clearer.
func (i *Instances) NodeAddresses(name types.NodeName) ([]api.NodeAddress, error) {
	glog.V(2).Infof("Instances.NodeAddresses() called with %s", name)

	ip, err := i.probeNodeAddress(string(name))
	if err != nil {
		glog.Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	parsedIP := net.ParseIP(ip).String()
	return []api.NodeAddress{
		{
			Type:    api.NodeLegacyHostIP,
			Address: parsedIP,
		},
		{
			Type:    api.NodeInternalIP,
			Address: parsedIP,
		},
		{
			Type:    api.NodeExternalIP,
			Address: parsedIP,
		},
	}, nil
}

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
//
// Note: If the instance does not exist or is no longer running, we must return
// ("", cloudprovider.InstanceNotFound)
func (i *Instances) ExternalID(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.ExternalID() called with %s", name)
	return i.probeMachineUUID(string(name))
}

// InstanceID returns the cloud provider ID of the node with the specified
// NodeName.
func (i *Instances) InstanceID(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.InstanceID() called with %s", name)
	return i.probeMachineUUID(string(name))
}

// InstanceType returns the type of the specified instance.
func (i *Instances) InstanceType(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.InstanceType() called with %s", name)
	return i.probeMachineBrand(string(name))
}

// List lists instances that match 'filter' which is a regular expression which
// must match the entire instance name (fqdn)
func (i *Instances) List(filter string) ([]types.NodeName, error) {
	glog.V(2).Infof("Instances.List() called with %s", filter)

	input := &triton.ListMachinesInput{}
	machines, err := i.provider.Client.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	names := make([]types.NodeName, len(machines))
	for _, machine := range machines {
		names = append(names, types.NodeName(machine.ID))
	}
	return names, nil
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all
// instances expected format for the key is standard ssh-keygen format:
// <protocol> <blob>
func (i *Instances) AddSSHKeyToAllInstances(user string, keyData []byte) error {
	return errors.New("unimplemented")
}

// CurrentNodeName returns the name of the node we are currently running on On
// most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (i *Instances) CurrentNodeName(hostname string) (types.NodeName, error) {
	glog.V(2).Infof("Instances.CurrentNodeName() called with %s", hostname)
	machine, err := i.getMachineByHostname(hostname)
	if err != nil {
		glog.Errorf("Instances.CurrentNodeName() returned an error: %s", err)
		return "", err
	}
	return types.NodeName(machine.ID), nil
}
