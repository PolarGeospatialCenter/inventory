package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	"github.com/spf13/viper"
)

type AnsibleGroup struct {
	Hosts []string               `json:"hosts"`
	Vars  map[string]interface{} `json:"vars"`
}

type AnsibleGroupList map[string]*AnsibleGroup

func (l *AnsibleGroupList) Get(groupname string) *AnsibleGroup {
	group, ok := (*l)[groupname]
	if !ok {
		log.Printf("Group doesn't exist yet, creating: %s", groupname)
		group = &AnsibleGroup{Hosts: []string{}, Vars: make(map[string]interface{})}
		(*l)[groupname] = group
	}
	log.Print(group)
	return group
}

func (g *AnsibleGroup) AddHost(hostname string) {
	g.Hosts = append(g.Hosts, hostname)
}

func main() {

	cfg := viper.New()
	cfg.AddConfigPath(".")
	cfg.SetConfigName("pgc-inventory")
	cfg.ReadInConfig()

	log.Printf("Loading filestore from: %s", cfg.GetString("path"))
	store, err := inventory.NewFileStore(cfg.GetString("path"))
	if err != nil {
		log.Fatalf("Unable to create file store: %v", err)
	}

	nodes, err := store.Nodes()
	if err != nil {
		log.Fatalf("Unable to read nodes: %v", err)
	}

	groups := make(AnsibleGroupList)
	hostVars := make(map[string]map[string]interface{})

	for _, node := range nodes {
		domain := node.Networks["provisioning"].Network.Domain
		fqdn := fmt.Sprintf("%s.%s", node.Hostname, domain)
		group := groups.Get(node.System.ID())
		group.AddHost(fqdn)
		roleGroup := fmt.Sprintf("%s-%s", node.System.ID(), node.Role)
		group = groups.Get(roleGroup)
		groups[roleGroup].AddHost(fqdn)
		hostVars[fqdn] = make(map[string]interface{})
		hostVars[fqdn]["ansible_python_interpreter"] = "/opt/ansible/bin/python"
		hostVars[fqdn]["tags"] = node.Tags
		hostVars[fqdn]["inventory_id"] = node.ID()
		hostVars[fqdn]["rack"] = node.Location.Rack
		hostVars[fqdn]["role"] = node.Role
		hostVars[fqdn]["ansible_host"] = node.Networks["provisioning"].NIC.IP
		hostVars[fqdn]["last_update"] = node.LastUpdated
		hostVars[fqdn]["nodeconfig"] = node
		if cpNetworkName, ok := node.Environment.Metadata["kubernetes_control_plane_network"].(string); ok {
			if cpNetwork, ok := node.Networks[cpNetworkName]; ok {
				hostVars[fqdn]["kube_control_plane_domain"] = cpNetwork.Network.Domain
				hostVars[fqdn]["kube_control_plane_ips"] = make([]string, 0, len(cpNetwork.Config.IP))
				for _, ipString := range cpNetwork.Config.IP {
					ip, _, err := net.ParseCIDR(ipString)
					if err == nil {
						hostVars[fqdn]["kube_control_plane_ips"] = append(hostVars[fqdn]["kube_control_plane_ips"].([]string), ip.String())
					}
				}
			}
		}
	}

	result := make(map[string]interface{})
	for gName, group := range groups {
		result[gName] = group
	}
	result["_meta"] = make(map[string]interface{})
	result["_meta"].(map[string]interface{})["hostvars"] = hostVars

	txt, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("Unable to marshal group data: %v", err)
	}
	fmt.Printf(string(txt))
}
