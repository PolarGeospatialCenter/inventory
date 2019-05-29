package dynamodbclient

import (
	"fmt"
	"net"
	"time"
)

type NodeMacIndexEntry struct {
	Mac         net.HardwareAddr
	LastUpdated time.Time
	NodeID      string
}

func (i *NodeMacIndexEntry) ID() string {
	return i.Mac.String()
}

func (i *NodeMacIndexEntry) Timestamp() int64 {
	return i.LastUpdated.Unix()
}

func (i *NodeMacIndexEntry) SetTimestamp(timestamp time.Time) {
	i.LastUpdated = timestamp
}

type nodeMacIndexStore struct {
	*DynamoDBStore
}

func (db *nodeMacIndexStore) GetMacIndexEntriesByNodeID(id string) (map[string]*NodeMacIndexEntry, error) {
	allMacs := make([]*NodeMacIndexEntry, 0, 0)
	err := db.getAll(&allMacs)
	if err != nil {
		return nil, fmt.Errorf("unable to get all NodeMacIndexEntries: %v", err)
	}

	results := make(map[string]*NodeMacIndexEntry, 0)
	for _, nodeMacIndexEntry := range allMacs {
		if nodeMacIndexEntry.NodeID == id {
			results[nodeMacIndexEntry.Mac.String()] = nodeMacIndexEntry
		}
	}
	return results, nil
}

func (db *nodeMacIndexStore) Create(e *NodeMacIndexEntry) error {
	return db.DynamoDBStore.create(e)
}

func (db *nodeMacIndexStore) Delete(e *NodeMacIndexEntry) error {
	return db.DynamoDBStore.delete(e)
}
