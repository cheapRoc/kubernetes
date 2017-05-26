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
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/golang/glog"
	gcfg "gopkg.in/gcfg.v1"
	"k8s.io/kubernetes/pkg/cloudprovider"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
)

const (
	ProviderName = "triton"
	KeyPath      = "/etc/triton/api_key"
)

type Triton struct {
	provider *triton.Client
}

type Config struct {
	Global struct {
		KeyID       string `gcfg:"key-id"`
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

// TODO: Probably can load key out of `mdata-get`, but for now its a
// requirement.

// newTriton constructs a new Triton object with our client as it's provider
func newTriton(cfg Config) (Triton, error) {
	privateKey, err := ioutil.ReadFile(KeyPath)
	if err != nil {
		glog.V(2).Error("newTriton() could not access KeyPath")
		return nil, err
	}

	sshKeySigner, err := authentication.NewPrivateKeySigner(cfg.Global.KeyID, privateKey,
		cfg.Global.AccountName)
	if err != nil {
		log.Fatal(err)
	}

	client, err := triton.NewClient(cfg.Global.Endpoint, cfg.Global.AccountName, sshKeySigner)
	if err != nil {
		log.Fatalf("NewClient: %s", err)
	}

	return &Triton{
		Provider: client,
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
