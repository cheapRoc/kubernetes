package triton

import (
	"context"
	"fmt"

	triton "github.com/joyent/triton-go"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/types"
)

type Instances struct {
	machines []triton.Machine
	provider triton.Client
}

// machineByName pulls the triton.Machine for a given types.NodeName
//
// TODO: Add functionality to refresh Instances.machines if a name isn't found
// within them. We could also pull out the storage of machines entirely... since
// they're bound to not be accurate or what we want over the long term.
func (i *Instances) byNodeName(name types.NodeName) (triton.Machine, error) {
	for _, inst := range i.machines {
		if inst.ID == name {
			return inst
		}
	}
	err := fmt.Errorf("machineByNodeName() could not find machine named: %s", name)
	return nil, err
}

// Instances returns all known cloud providers instances running for our
// configured Triton client.
func (t *Triton) Instances() (cloudprovider.Instances, bool) {
	glog.V(2).Info("Triton.Instances() called")

	input := &triton.ListMachinesInput{}
	machines, err := t.provider.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, false
	}

	return &Instances{
		machines: machines,
		provider: triton.Provider,
	}, true
}

// NodeAddresses returns the addresses of the specified instance.
//
// NOTE: This currently is only used in such a way that it returns the address
// of the calling instance. We should do a rename to make this clearer.
func (i *Instances) NodeAddresses(name types.NodeName) ([]api.NodeAddress, error) {
	glog.V(2).Infof("Instances.NodeAddresses() called with %s", name)

	inst, err := i.byNodeName(name)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	input := triton.GetMachineInput{inst.ID}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, err
	}

	addresses := make([]api.NodeAddress, len(machine.IPs))
	for _, ipAddress := range machine.IPs {
		address := api.NodeAddress{api.NodeExternalIP, ipAddress}
		addresses = append(addresses, address)
	}
	return addresses, nil
}

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
//
// Note: If the instance does not exist or is no longer running, we must return
// ("", cloudprovider.InstanceNotFound)
func (i *Instances) ExternalID(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.ExternalID() called with %s", name)

	inst, err := i.byNodeName(name)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, cloudprovider.InstanceNotFound
	}

	input := triton.GetMachineInput{inst.ID}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, cloudprovider.InstanceNotFound
	}
	if machine.State != "running" {
		glog.V(2).Errorf("Machine.State is not running, returned: %s", machine.State)
		return nil, cloudprovider.InstanceNotFound
	}

	return machine.PrimaryIP
}

// InstanceID returns the cloud provider ID of the node with the specified
// NodeName.
func (i *Instances) InstanceID(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.InstanceID() called with %s", name)

	inst, err := i.machineByName(name)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	input := triton.GetMachineInput{inst.ID}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, err
	}

	return machine.ID, nil
}

// InstanceType returns the type of the specified instance.
func (i *Instances) InstanceType(name types.NodeName) (string, error) {
	glog.V(2).Infof("Instances.InstanceType() called with %s", name)

	inst, err := i.byNodeName(name)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	input := triton.GetMachineInput{inst.ID}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, err
	}

	return machine.Brand, nil
}

// List lists instances that match 'filter' which is a regular expression which
// must match the entire instance name (fqdn)
func (i *Instances) List(filter string) ([]types.NodeName, error) {
	glog.V(2).Infof("Instances.List() called with %s", filter)

	input := &triton.ListMachinesInput{}
	machines, err := i.Provider.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, false
	}

	names := make([]types.NodeName, len(machines))
	for _, machine := range machines {
		names = append(names, types.NodeName{machine.ID})
	}
	return names
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all
// instances expected format for the key is standard ssh-keygen format:
// <protocol> <blob>
// func (i *Instances) AddSSHKeyToAllInstances(user string, keyData []byte) error {

// }

// CurrentNodeName returns the name of the node we are currently running on On
// most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (i *Instances) CurrentNodeName(hostname string) (types.NodeName, error) {
	glog.V(2).Infof("Instances.CurrentNodeName() called with %s", hostname)

	return types.NodeName{hostname}, nil
}
