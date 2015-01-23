/**
 * Copyright 2014 Rackspace, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

const IRONIC_API_VERSION = "v1"
const LOOKUP_PAYLOAD_VERSION = "2"

var DEBUG = false

type LookupPayload struct {
	Version   string            `json:"version"`
	Inventory HardwareInventory `json:"inventory"`
}

type HardwareInventory struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
}

type InterfaceInfo struct {
	Name       string `json:"name"`
	MacAddress string `json:"mac_address"`
}

func InterfaceIsDevice(iface net.Interface) (bool, error) {
	_, err := os.Stat("/sys/class/net/" + iface.Name + "/device")
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

func BuildLookupPayload() (*LookupPayload, error) {
	interfaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	interfaceInfos := make([]InterfaceInfo, 0)

	for _, iface := range interfaces {
		isDevice, err := InterfaceIsDevice(iface)
		if err != nil {
			return nil, err
		}

		if isDevice {
			interfaceInfos = append(interfaceInfos, InterfaceInfo{
				Name:       iface.Name,
				MacAddress: iface.HardwareAddr.String(),
			})
		}
	}

	payload := &LookupPayload{
		Version: LOOKUP_PAYLOAD_VERSION,
		Inventory: HardwareInventory{
			Interfaces: interfaceInfos,
		},
	}

	return payload, nil
}

type IronicAPIClient struct {
	URL        string
	DriverName string
	client     *http.Client
}

func NewAPIClient(url string, driverName string) *IronicAPIClient {
	// Canonicalize the URL to have a trailing slash, just because
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	return &IronicAPIClient{
		URL:        url,
		DriverName: driverName,
		client:     &http.Client{},
	}
}

type IronicNode struct {
	UUID         string `json:"uuid"`
	InstanceInfo struct {
		RescuePasswordHash string `json:"rescue_password_hash"`
		ConfigDrive        string `json:"configdrive"`
	} `json:"instance_info"`
}

type LookupResponse struct {
	Node IronicNode `json:"node"`
}

func (c *IronicAPIClient) do(method string, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, c.URL+IRONIC_API_VERSION+path, bodyReader)

	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	req.Header.Add("Accept", "application/json")

	return c.client.Do(req)
}

func (c *IronicAPIClient) Lookup(payload *LookupPayload) (*IronicNode, error) {
	res, err := c.do("POST", "/drivers/"+c.DriverName+"/vendor_passthru/lookup", payload)
	if err != nil {
		return nil, err
	}

	// TODO: Some kind of retry
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("Unexpected response from Ironic lookup call: " + res.Status)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	lookupResponse := LookupResponse{}
	if err := json.Unmarshal(body, &lookupResponse); err != nil {
		return nil, err
	}

	return &lookupResponse.Node, nil
}

func (c *IronicAPIClient) Heartbeat(uuid string) error {
	payload := map[string]string{
		"agent_url": "",
	}

	res, err := c.do("POST", "/nodes/"+uuid+"/vendor_passthru/heartbeat", payload)
	if err != nil {
		return err
	}

	// TODO: Some kind of retry
	if res.StatusCode != http.StatusAccepted {
		return errors.New("Unexpected response from Ironic heartbeat call: " + res.Status)
	}

	return nil
}

func main() {
	var apiURL = ""

	flag.BoolVar(&DEBUG, "debug", false, "Debug mode")
	flag.StringVar(&apiURL, "api-url", "", "Ironic API URL")
	flag.Parse()

	log.Print(DEBUG)
	if apiURL == "" {
		flag.Parse()
		log.Fatal("No Ironic API URL specified")
	}

	c := NewAPIClient(apiURL, "agent_ipmitool")

	payload, err := BuildLookupPayload()
	if err != nil {
		log.Fatal("Error building lookup payload: ", err)
	}
	if DEBUG {
		log.Print(payload)
	}

	node, err := c.Lookup(payload)
	if err != nil {
		log.Fatal("Error in lookup call: ", err)
	}
	if DEBUG {
		log.Print(node.UUID)
		log.Print(node.InstanceInfo.ConfigDrive)
		log.Print(node.InstanceInfo.RescuePasswordHash)
	}

	err = c.Heartbeat(node.UUID)
	if err != nil {
		log.Fatal("Error in heartbeat: ", err)
	}
}
