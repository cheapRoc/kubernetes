package triton

import (
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	_, err := readConfig(nil)
	if err == nil {
		t.Errorf("Should fail when no config is provided: %s", err)
	}

	testConfig := strings.NewReader(`
[Global]
endpoint-url = https://us-sw-1.api.joyent.com
key-id = 95:ec:59:3d:73:a8:ae:6b:d0:ec:21:d7:6e:e9:f5:6e
key-path = /etc/kubernetes/api_key
account = testuser
`)

	cfg, err := readConfig(testConfig)
	if err != nil {
		t.Fatalf("Should be parsed when a valid config is provided: %s", err)
	}
	if cfg.Global.EndpointURL != "https://us-sw-1.api.joyent.com" {
		t.Errorf("Should fail when can't match endpoint-url: %s", cfg.Global.EndpointURL)
	}
	if cfg.Global.KeyID != "95:ec:59:3d:73:a8:ae:6b:d0:ec:21:d7:6e:e9:f5:6e" {
		t.Errorf("Should fail when can't match key-id: %s", cfg.Global.KeyID)
	}
	if cfg.Global.AccountName != "testuser" {
		t.Errorf("Should fail when can't match account: %s", cfg.Global.AccountName)
	}
	if cfg.Global.KeyPath != "/etc/kubernetes/api_key" {
		t.Errorf("Should fail when can't match key-id: %s", cfg.Global.KeyID)
	}
}
