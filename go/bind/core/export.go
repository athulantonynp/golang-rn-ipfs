package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"berty.tech/go-orbit-db/iface"
	ipfs_node "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ipfsmobile"
)

type OrbitDb struct{
	db iface.EventLogStore
	events []string
 	eventsMutex sync.Mutex
}

func NewOrbitDB()(*OrbitDb){
	var odb = OrbitDb{}
	odb.db = ipfs_node.GetOrbitDb()
	return &odb
}

func(ob *OrbitDb) StartSubscription() {
	go func (){
		var ctx = context.Background()
		eventsChan := ob.db.Subscribe(ctx)
	for {
		select {
		case event := <-eventsChan:
			var e Event
			log.Println("Received from channel eventsChan:", event)
			eventJSON, err := json.Marshal(event)
			if err != nil {
				fmt.Println("Error marshalling event:", err)
				continue
			}

			err = json.Unmarshal(eventJSON, &e)
			if err != nil {
				fmt.Println("Error unmarshalling event:", err)
				continue
			}

			payloadBytes, err := base64.StdEncoding.DecodeString(e.Entry.Payload)
			if err != nil {
				fmt.Println("Error decoding payload:", err)
				continue
			}

			payload := string(payloadBytes)

			var pt PayloadType

			er := json.Unmarshal([]byte(payload), &pt)
			if er != nil {
				fmt.Println("Error unmarshalling payload:", er)
				continue
			}

			decodedValue, err := base64.StdEncoding.DecodeString(pt.Value)
			if err != nil {
				fmt.Println("Error decoding payload:", err)
				continue
			}

			ob.eventsMutex.Lock()
			ob.events = append(ob.events, string(decodedValue))
			ob.eventsMutex.Unlock()
			log.Println("Received event:", e.Entry, pt, e.Address, string(decodedValue))

		case <-ctx.Done():
			return
		}
	}
	}()
}

func (ob *OrbitDb) SendEvents( buffer []byte) {
	_, err := ob.db.Add(context.Background(), []byte(buffer))
	if err != nil {
		log.Println("Error adding event to database:", err)
	}
}

func(ob *OrbitDb) StopSubscription() {
	if ob.db != nil {
		ob.db.Close()
	}
}

func(ob *OrbitDb) GetEvents() []string {
	ob.eventsMutex.Lock()
	defer ob.eventsMutex.Unlock()

	// Create a copy of the events slice
	eventsCopy := make([]string, len(ob.events))
	copy(eventsCopy, ob.events)
	log.Println("Events:", eventsCopy)

	return eventsCopy
}
