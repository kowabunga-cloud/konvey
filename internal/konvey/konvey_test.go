/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package konvey

import (
	"os"
	"strings"
	"testing"

	"github.com/kowabunga-cloud/common/agents"
	"github.com/kowabunga-cloud/common/agents/templates"
)

// newTestKonveyServices returns a fresh services map for each test.
// AgentTestTemplate mutates the map in-place (prefixes paths, sets user/group),
// so each test must start with its own copy.
func newTestKonveyServices() map[string]*agents.ManagedService {
	return map[string]*agents.ManagedService{
		"keepalived": {
			BinaryPath: "",
			UnitName:   "keepalived",
			ConfigPaths: []agents.ConfigFile{
				{
					TemplateContent: templates.KeepalivedConfTemplate("konvey"),
					TargetPath:      "keepalived.conf",
				},
			},
		},
		"traefik": {
			BinaryPath: "",
			UnitName:   "traefik",
			ConfigPaths: []agents.ConfigFile{
				{
					TemplateContent: templates.TraefikConfTemplate("konvey"),
					TargetPath:      "traefik.yml",
				},
				{
					TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "tcp"),
					TargetPath:      "tcp.yml",
				},
				{
					TemplateContent: templates.TraefikLayer4ConfTemplate("konvey", "udp"),
					TargetPath:      "udp.yml",
				},
			},
		},
	}
}

var testKonveyConfig = map[string]any{
	"konvey": map[string]any{
		"private_interface":      "ens4",
		"vrrp_control_interface": "ens4",
		"virtual_ips": []map[string]any{
			{
				"vrrp_id":   1,
				"interface": "ens4",
				"vip":       "192.168.0.10",
				"priority":  100,
				"mask":      24,
				"public":    false,
			},
		},
		"endpoints": []map[string]any{
			{
				"name":     "proxyServer",
				"port":     8080,
				"protocol": "tcp",
				"backends": []map[string]any{
					{
						"host": "192.168.0.20",
						"port": 8080,
					},
					{
						"host": "192.168.0.21",
						"port": 8080,
					},
				},
			},
		},
	},
}

// TestKonveyServicesDefinition verifies the production konveyServices map has the correct
// unit names, users, groups, config path counts, and target paths.
func TestKonveyServicesDefinition(t *testing.T) {
	if _, ok := konveyServices["keepalived"]; !ok {
		t.Fatal("konveyServices missing 'keepalived' entry")
	}
	if _, ok := konveyServices["traefik"]; !ok {
		t.Fatal("konveyServices missing 'traefik' entry")
	}

	ka := konveyServices["keepalived"]
	if ka.UnitName != "keepalived.service" {
		t.Errorf("keepalived UnitName = %q, want %q", ka.UnitName, "keepalived.service")
	}
	if ka.User != "root" {
		t.Errorf("keepalived User = %q, want %q", ka.User, "root")
	}
	if ka.Group != "root" {
		t.Errorf("keepalived Group = %q, want %q", ka.Group, "root")
	}
	if len(ka.ConfigPaths) != 1 {
		t.Fatalf("keepalived ConfigPaths len = %d, want 1", len(ka.ConfigPaths))
	}
	if ka.ConfigPaths[0].TargetPath != "/etc/keepalived/keepalived.conf" {
		t.Errorf("keepalived config path = %q, want %q", ka.ConfigPaths[0].TargetPath, "/etc/keepalived/keepalived.conf")
	}

	tr := konveyServices["traefik"]
	if tr.UnitName != "traefik.service" {
		t.Errorf("traefik UnitName = %q, want %q", tr.UnitName, "traefik.service")
	}
	if tr.User != "traefik" {
		t.Errorf("traefik User = %q, want %q", tr.User, "traefik")
	}
	if tr.Group != "traefik" {
		t.Errorf("traefik Group = %q, want %q", tr.Group, "traefik")
	}
	if len(tr.ConfigPaths) != 3 {
		t.Fatalf("traefik ConfigPaths len = %d, want 3", len(tr.ConfigPaths))
	}
	wantPaths := []string{
		"/etc/traefik/traefik.yml",
		"/etc/traefik/conf.d/tcp.yml",
		"/etc/traefik/conf.d/udp.yml",
	}
	for i, want := range wantPaths {
		if tr.ConfigPaths[i].TargetPath != want {
			t.Errorf("traefik ConfigPaths[%d] = %q, want %q", i, tr.ConfigPaths[i].TargetPath, want)
		}
	}
}

// TestKonveyTemplate tests that template rendering with a standard config succeeds.
func TestKonveyTemplate(t *testing.T) {
	agents.AgentTestTemplate(t, newTestKonveyServices(), t.TempDir(), testKonveyConfig)
}

// TestKonveyTemplateRenderedKeepalivedContent verifies keepalived.conf contains expected values.
func TestKonveyTemplateRenderedKeepalivedContent(t *testing.T) {
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, testKonveyConfig)

	content, err := os.ReadFile(dir + "/keepalived.conf")
	if err != nil {
		t.Fatalf("failed to read keepalived.conf: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		"vrrp_instance VI_1",
		"interface ens4",
		"virtual_router_id 1",
		"priority 100",
		"192.168.0.10/24 dev ens4",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("keepalived.conf missing %q\nContent:\n%s", want, s)
		}
	}
}

// TestKonveyTemplateRenderedTraefikContent verifies traefik.yml contains expected values.
func TestKonveyTemplateRenderedTraefikContent(t *testing.T) {
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, testKonveyConfig)

	content, err := os.ReadFile(dir + "/traefik.yml")
	if err != nil {
		t.Fatalf("failed to read traefik.yml: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		"checkNewVersion: false",
		"sendAnonymousUsage: false",
		"proxyServer:",
		":8080/tcp",
		"directory: \"/etc/traefik/conf.d\"",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("traefik.yml missing %q\nContent:\n%s", want, s)
		}
	}
}

// TestKonveyTemplateRenderedTCPContent verifies tcp.yml is rendered with expected routers and backends.
func TestKonveyTemplateRenderedTCPContent(t *testing.T) {
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, testKonveyConfig)

	content, err := os.ReadFile(dir + "/tcp.yml")
	if err != nil {
		t.Fatalf("failed to read tcp.yml: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		"tcp:",
		"proxyServer:",
		"HostSNI(`*`)",
		"192.168.0.20:8080",
		"192.168.0.21:8080",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("tcp.yml missing %q\nContent:\n%s", want, s)
		}
	}
}

// TestKonveyTemplateEmptyConfig verifies rendering with no virtual_ips or endpoints does not fail.
func TestKonveyTemplateEmptyConfig(t *testing.T) {
	cfg := map[string]any{
		"konvey": map[string]any{
			"private_interface":      "ens4",
			"vrrp_control_interface": "ens4",
			"virtual_ips":            []map[string]any{},
			"endpoints":              []map[string]any{},
		},
	}
	agents.AgentTestTemplate(t, newTestKonveyServices(), t.TempDir(), cfg)
}

// TestKonveyTemplatePublicVIP verifies that a VIP with public=true includes a virtual_routes block.
func TestKonveyTemplatePublicVIP(t *testing.T) {
	cfg := map[string]any{
		"konvey": map[string]any{
			"private_interface":      "ens4",
			"vrrp_control_interface": "ens4",
			"public_interface":       "ens5",
			"public_gw_address":      "10.0.0.1",
			"virtual_ips": []map[string]any{
				{
					"vrrp_id":   2,
					"interface": "ens5",
					"vip":       "10.0.0.50",
					"priority":  100,
					"mask":      24,
					"public":    true,
				},
			},
			"endpoints": []map[string]any{},
		},
	}
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, cfg)

	content, err := os.ReadFile(dir + "/keepalived.conf")
	if err != nil {
		t.Fatalf("failed to read keepalived.conf: %v", err)
	}
	s := string(content)

	for _, want := range []string{
		"virtual_routes",
		"0.0.0.0/0 via 10.0.0.1 dev ens5",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("keepalived.conf missing %q for public VIP\nContent:\n%s", want, s)
		}
	}
}

// TestKonveyTemplateMultipleVRRPInstances verifies that multiple VRRP IDs produce separate vrrp_instance blocks.
func TestKonveyTemplateMultipleVRRPInstances(t *testing.T) {
	cfg := map[string]any{
		"konvey": map[string]any{
			"private_interface":      "ens4",
			"vrrp_control_interface": "ens4",
			"virtual_ips": []map[string]any{
				{
					"vrrp_id":   1,
					"interface": "ens4",
					"vip":       "192.168.0.10",
					"priority":  100,
					"mask":      24,
					"public":    false,
				},
				{
					"vrrp_id":   2,
					"interface": "ens4",
					"vip":       "192.168.0.11",
					"priority":  90,
					"mask":      24,
					"public":    false,
				},
			},
			"endpoints": []map[string]any{},
		},
	}
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, cfg)

	content, err := os.ReadFile(dir + "/keepalived.conf")
	if err != nil {
		t.Fatalf("failed to read keepalived.conf: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "vrrp_instance VI_1") {
		t.Errorf("keepalived.conf missing vrrp_instance VI_1\nContent:\n%s", s)
	}
	if !strings.Contains(s, "vrrp_instance VI_2") {
		t.Errorf("keepalived.conf missing vrrp_instance VI_2\nContent:\n%s", s)
	}
}

// TestKonveyTemplateMultipleEndpoints verifies multiple endpoints render to the correct protocol files.
func TestKonveyTemplateMultipleEndpoints(t *testing.T) {
	cfg := map[string]any{
		"konvey": map[string]any{
			"private_interface":      "ens4",
			"vrrp_control_interface": "ens4",
			"virtual_ips":            []map[string]any{},
			"endpoints": []map[string]any{
				{
					"name":     "httpProxy",
					"port":     80,
					"protocol": "tcp",
					"backends": []map[string]any{
						{"host": "10.0.0.1", "port": 80},
					},
				},
				{
					"name":     "dnsProxy",
					"port":     53,
					"protocol": "udp",
					"backends": []map[string]any{
						{"host": "10.0.0.2", "port": 53},
					},
				},
			},
		},
	}
	dir := t.TempDir()
	agents.AgentTestTemplate(t, newTestKonveyServices(), dir, cfg)

	traefikContent, err := os.ReadFile(dir + "/traefik.yml")
	if err != nil {
		t.Fatalf("failed to read traefik.yml: %v", err)
	}
	ts := string(traefikContent)
	for _, want := range []string{"httpProxy:", ":80/tcp", "dnsProxy:", ":53/udp"} {
		if !strings.Contains(ts, want) {
			t.Errorf("traefik.yml missing %q\nContent:\n%s", want, ts)
		}
	}

	tcpContent, err := os.ReadFile(dir + "/tcp.yml")
	if err != nil {
		t.Fatalf("failed to read tcp.yml: %v", err)
	}
	if !strings.Contains(string(tcpContent), "httpProxy:") {
		t.Errorf("tcp.yml missing httpProxy entry\nContent:\n%s", string(tcpContent))
	}

	udpContent, err := os.ReadFile(dir + "/udp.yml")
	if err != nil {
		t.Fatalf("failed to read udp.yml: %v", err)
	}
	if !strings.Contains(string(udpContent), "dnsProxy:") {
		t.Errorf("udp.yml missing dnsProxy entry\nContent:\n%s", string(udpContent))
	}
}
