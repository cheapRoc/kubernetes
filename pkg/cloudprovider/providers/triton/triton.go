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

type Client struct {
	*triton.Client
}

type Config struct {
	Client struct {
		keyID       string `gcfg:"key-id"`
		endpoint    string `gcfg:"endpoint"`
		accountName string `gcfg:"account"`
	}
}

type Provider struct {
	client Client
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := readConfig(config)
		if err != nil {
			return nil, err
		}
		return newTriton(cfg)
	})
}

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
func newTriton(cfg Config) (Client, error) {
	privateKey, err := ioutil.ReadFile(KeyPath)
	if err != nil {
		glog.V(2).Error("newTriton() could not access KeyPath")
		return err
	}

	sshKeySigner, err := authentication.NewPrivateKeySigner(cfg.keyID, cfg.privateKey, cfg.accountName)
	if err != nil {
		log.Fatal(err)
	}

	client, err := triton.NewClient(cfg.endpoint, cfg.accountName, sshKeySigner)
	if err != nil {
		log.Fatalf("NewClient: %s", err)
	}

	return &Client{client}
}
