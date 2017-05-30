package triton

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/golang/glog"
	triton "github.com/joyent/triton-go"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/types"
)

type Instances struct {
	machines []triton.Machine
	provider triton.Client
}

// grab UUID of our machine's instance from Joyent's Metadata utility
// `mdata-get`
func readMetadataUUID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/sbin/mdata-get", "sdc:uuid").Output()
	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}

// getMachineByUUID returns the triton.Machine for a given UUID
func (i Instances) getMachineByUUID(uuid string) triton.Machine {
	input := triton.GetMachineInput{uuid}
	machine, err := i.provider.Client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Machines.GetMachine() returned an error: %s", err)
		return nil, err
	}
	return machine, nil
}

// probeNodeAddress returns the IP address for the current metadata UUID or a
// given serverName.
func (i Instances) probeNodeAddress(serverName string) (string, error) {
	uuid, err := readMetadataUUID()
	if err == nil {
		machine, err := i.getMachineByUUID(uuid)
		if err != nil {
			return "", err
		}
		return machine.PrimaryIP, nil
	}

	machine, err := i.probeMachineUUID(serverName)
	if err != nil {
		return nil, err
	}

	return machine.PrimaryIP, nil
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

	ip, err := i.probeNodeAddress(string(serverName))
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
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

// getMachineByName searches through all machines within a Triton account
// looking for a match by passed in serverName string. Checks (within order)
// PrimaryIP, Hostname, and UUID.
func (i Instances) getMachineByName(serverName string) (triton.Machine, error) {
	input := &triton.ListMachinesInput{}
	machines, err := i.Provider.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, err
	}

	for _, machine := range machines {
		if machine.PrimaryIP == serverName {
			return machine
		}
		if machine.Hostname == serverName {
			return machine
		}
		if machine.ID == serverName {
			return machine
		}
	}

	return nil, fmt.Errorf("No machine found by serverName: %s", serverName)
}

// probeMachineUUID probes the existing machine's metadata for a UUID and
// returns, otherwise we search through all machines and return an ID that way.
func (i Instances) probeMachineUUID(serverName string) (string, error) {
	id, err := readMachineUUID()
	if err == nil {
		return id, nil
	}

	machine, err := i.getMachineByName(serverName)
	if err != nil {
		return "", nil
	}
	return machine.ID, nil
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

// probeNodeAddress returns the IP address for the current metadata UUID or a
// given serverName.
func (i Instances) probeMachineBrand(serverName string) (string, error) {
	uuid, err := readMetadataUUID()
	if err == nil {
		machine, err := i.getMachineByUUID(uuid)
		if err != nil {
			return "", err
		}
		return machine.Brand, nil
	}

	machine, err := i.probeMachineUUID(serverName)
	if err != nil {
		return nil, err
	}

	return machine.PrimaryIP, nil
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
	machines, err := i.provider.Machines().ListMachines(context.Background(), input)
	if err != nil {
		glog.V(2).Errorf("Triton.Instances() returned an error: %s", err)
		return nil, false
	}

	names := make([]types.NodeName, len(machines))
	for _, machine := range machines {
		names = append(names, types.NodeName(machine.ID))
	}
	return names
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
	return types.NodeName(hostname), nil
}
