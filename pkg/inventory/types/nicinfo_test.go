package types

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func getTestNICInfo() (*NetworkInterface, string) {
	mac, _ := net.ParseMAC("00:02:03:04:05:06")
	nicInfo := &NetworkInterface{NICs: []net.HardwareAddr{mac}, Metadata: Metadata{}}
	marshaledJSON := `{"Metadata":{},"nics":["00:02:03:04:05:06"]}`
	return nicInfo, marshaledJSON
}

func TestNICInfoMarshal(t *testing.T) {
	i, expected := getTestNICInfo()
	result, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nic info %v", i)
	}

	if string(result) != expected {
		t.Fatalf("The marshaled version of the NICInfo is incorrect: '%s', Expected: '%s'", string(result), expected)
	}

	i = &NetworkInterface{}
	result, err = json.Marshal(i)
	if err != nil {
		t.Fatalf("Unable to marshal nil nic info: %v", err)
	}

	if string(result) != `{"Metadata":{},"nics":[]}` {
		t.Fatalf("Marshaled version of nil NICInfo incorrect: %s", string(result))
	}
}

func TestNICInfoUnmarshalJSON(t *testing.T) {
	expected, testText := getTestNICInfo()
	info := &NetworkInterface{}
	testUnmarshalJSON(t, info, expected, testText)
}

func TestLegacyNICInfoDynamoDBUnmarshal(t *testing.T) {
	mac, _ := net.ParseMAC("00:01:02:03:04:05")
	ni := &NICInfo{MAC: mac}
	marshaledNicInfo, err := dynamodbattribute.Marshal(ni)
	if err != nil {
		t.Errorf("error marshaling legacy nicinfo object: %v", err)
	}

	iface := &NetworkInterface{}
	err = dynamodbattribute.Unmarshal(marshaledNicInfo, iface)
	if err != nil {
		t.Errorf("unable to unmarshal legacy nicinfo to network interface: %v", err)
	}

	if len(iface.NICs) == 0 || iface.NICs[0].String() != ni.MAC.String() {
		t.Errorf("not unmarshaled properly, got %v", iface)
	}
}

func TestLegacyNICInfoMapDynamoDBUnmarshal(t *testing.T) {
	mac, _ := net.ParseMAC("00:01:02:03:04:05")
	ni := &NICInfo{MAC: mac}
	niMap := map[string]*NICInfo{"testnet": ni}
	marshaledNicInfoMap, err := dynamodbattribute.Marshal(niMap)
	if err != nil {
		t.Errorf("error marshaling legacy nicinfo object: %v", err)
	}

	ifaceMap := NICInfoMap{}
	err = dynamodbattribute.Unmarshal(marshaledNicInfoMap, &ifaceMap)
	if err != nil {
		t.Errorf("unable to unmarshal legacy nicinfo to network interface: %v", err)
	}

	if len(ifaceMap) == 0 || len(ifaceMap["testnet"].NICs) == 0 || ifaceMap["testnet"].NICs[0].String() != ni.MAC.String() {
		t.Errorf("not unmarshaled properly, got %v", ifaceMap)
	}
}

func TestLegacyNICInfoMapDynamoDBUnmarshalEmptyMAC(t *testing.T) {
	mac, _ := net.ParseMAC("00:01:02:03:04:05")
	ni := &NICInfo{MAC: mac}
	niMap := map[string]*NICInfo{"testnet": ni}
	marshaledNicInfoMap, err := dynamodbattribute.Marshal(niMap)
	if err != nil {
		t.Errorf("error marshaling legacy nicinfo object: %v", err)
	}

	emptyMac, _ := dynamodbattribute.Marshal("")
	marshaledNicInfoMap.M["testnet"].M["MAC"] = emptyMac

	ifaceMap := NICInfoMap{}
	err = dynamodbattribute.Unmarshal(marshaledNicInfoMap, &ifaceMap)
	if err != nil {
		t.Errorf("unable to unmarshal legacy nicinfo to network interface: %v", err)
	}

	if len(ifaceMap) == 0 || len(ifaceMap["testnet"].NICs) != 0 {
		t.Errorf("not unmarshaled properly, got %v", ifaceMap)
	}
}
