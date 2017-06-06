/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package triton

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/golang/glog"
	gcfg "gopkg.in/gcfg.v1"
	"k8s.io/kubernetes/pkg/cloudprovider"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
)

const ProviderName = "triton"

type Triton struct {
	Client   *triton.Client
	Metadata *Metadata
	Instance *triton.Machine
}

type Metadata struct {
	UUID     string
	Hostname string
}

type Config struct {
	Global struct {
		KeyID       string `gcfg:"key-id"`
		KeyPath     string `gcfg:"key-path"`
		EndpointURL string `gcfg:"endpoint-url"`
		AccountName string `gcfg:"account"`
	}
}

// init registers and loads Triton as a cloud provider
func init() {
	cloudprovider.RegisterCloudProvider(ProviderName,
		func(config io.Reader) (cloudprovider.Interface, error) {
			cfg, err := readConfig(config)
			if err != nil {
				return nil, err
			}
			return newTriton(cfg)
		})
}

// readConfig reads our provider's configuration file
func readConfig(config io.Reader) (Config, error) {
	if config == nil {
		err := fmt.Errorf("no Triton cloud provider config file given")
		return Config{}, err
	}

	var cfg Config
	err := gcfg.ReadInto(&cfg, config)
	return cfg, err
}

// initMetadata returns a Metadata object initialized by shelling out the the
// `mdata-get` client.
//
// TODO: Right now this is mandatory because the rest of the API will require
// the host UUID.
func initMetadata() (*Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	uuid, err := exec.CommandContext(ctx, "/usr/sbin/mdata-get", "sdc:uuid").Output()
	if err != nil {
		return nil, err
	}

	var (
		hname string
		out   bytes.Buffer
	)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel2()
	cmd := exec.CommandContext(ctx2, "/usr/sbin/mdata-get", "sdc:hostname")
	cmd.Stdout = &out
	err = cmd.Run()
	if err == nil {
		hname = string(out.Bytes())
	} else {
		hname = string(uuid)
	}

	return &Metadata{
		UUID:     string(uuid),
		Hostname: string(hname),
	}, nil
}

// newTriton constructs a new Triton object with our client as it's provider
func newTriton(cfg Config) (*Triton, error) {
	privateKey, err := ioutil.ReadFile(cfg.Global.KeyPath)
	if err != nil {
		glog.Error("newTriton: could not access configured KeyPath")
		return nil, err
	}

	sshKeySigner, err := authentication.NewPrivateKeySigner(cfg.Global.KeyID, privateKey,
		cfg.Global.AccountName)
	if err != nil {
		log.Fatal(err)
	}

	client, err := triton.NewClient(cfg.Global.EndpointURL, cfg.Global.AccountName, sshKeySigner)
	if err != nil {
		log.Fatalf("NewClient: %s", err)
	}

	metadata, err := initMetadata()
	if err != nil {
		log.Fatalf("initMetadata: %s", err)
	}

	input := &triton.GetMachineInput{metadata.UUID}
	localhost, err := client.Machines().GetMachine(context.Background(), input)
	if err != nil {
		log.Fatalf("GetMachineInput for localhost: %s", err)
	}

	return &Triton{
		Client:   client,
		Metadata: metadata,
		Instance: localhost,
	}, nil
}

// ProviderName returns our ProviderName, which is hopefully always "triton"
func (t *Triton) ProviderName() string {
	return ProviderName
}

// ScrubDNS filters DNS settings for pods, giving us a chance to do interesting
// things with the configuration.
func (t *Triton) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nameservers, searches
}

// LoadBalancer is just a stub
func (t *Triton) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

// Clusters is just a stub
func (t *Triton) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Zones is just a stub
func (t *Triton) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// Routes is just a stub
func (t *Triton) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}
