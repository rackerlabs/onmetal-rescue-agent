package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServerAndClient(code int, body string) (*httptest.Server, *IronicAPIClient) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		fmt.Fprintln(w, body)
	}))
	client := &IronicAPIClient{
		server.URL + "/",
		"agent_ipmi",
		&http.Client{},
	}
	return server, client
}


func TestLookup(t *testing.T) {
	server, client := testServerAndClient(http.StatusOK, `{"node":{"uuid":"79e75f82-5a82-467f-90d5-bfbe2df39d15"}}`)
	defer server.Close()

	interfaceInfos := make([]InterfaceInfo, 0)
	interfaceInfos = append(interfaceInfos, InterfaceInfo{
			Name: "eth0",
			MacAddress: "aa:bb:cc:dd:ee:ff",
		})

	payload := &LookupPayload{
		Version: LOOKUP_PAYLOAD_VERSION,
		Inventory: HardwareInventory{
			Interfaces: interfaceInfos,
		},
	}

	node, err := client.Lookup(payload)
	if err != nil {
		log.Fatal("Error in lookup call: ", err)
	}

	if node.UUID != "79e75f82-5a82-467f-90d5-bfbe2df39d15" {
		t.Fail()
	}
}