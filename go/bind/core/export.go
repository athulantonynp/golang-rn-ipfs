package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"berty.tech/go-orbit-db/iface"
	"berty.tech/go-orbit-db/stores/replicator"
	ipfs_node "github.com/ipfs-shipyard/gomobile-ipfs/go/pkg/ipfsmobile"
)

type OrbitDb struct {
	db          iface.EventLogStore
	events      []string
	eventsMutex sync.Mutex
}

func NewOrbitDB() *OrbitDb {
	var odb = OrbitDb{}
	odb.db = ipfs_node.GetOrbitDb()
	return &odb
}

type MessageCallback interface {
	OnMessage(string)
}

func (ob *OrbitDb) StartSubscription(callback MessageCallback) {
	go func() {
		var ctx = context.Background()
		var repEvents = replicator.Events
		eventsChan := ob.db.Subscribe(ctx)
		repSub, err := ob.db.Replicator().EventBus().Subscribe(&repEvents)
		if err != nil {
			log.Println("Error subscribing to replicator event:", err)
			return
		}
		stop := false
		repChan := repSub.Out()
		for {
			log.Println("Replication status:", ob.db.ReplicationStatus().GetProgress(), "/", ob.db.ReplicationStatus().GetMax())
			if ob.db.ReplicationStatus().GetProgress() == ob.db.ReplicationStatus().GetMax() && !stop {
				ob.db.Load(ctx, ob.db.ReplicationStatus().GetMax())
				log.Println("Replication completed")
				stop = true
			}
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
				callback.OnMessage(string(decodedValue))
				log.Println("Received event:", e.Entry, pt, e.Address, string(decodedValue))
			case e := <-repChan:
				log.Println("Received from channel repChan:", e)

			case <-ctx.Done():
				return
			}

		}
	}()
}

func (ob *OrbitDb) SendEvents(buffer []byte) {
	_, err := ob.db.Add(context.Background(), []byte(buffer))
	if err != nil {
		log.Println("Error adding event to database:", err)
	}
}

func (ob *OrbitDb) StopSubscription() {
	if ob.db != nil {
		ob.db.Close()
	}
}

func (ob *OrbitDb) GetEvents() []string {
	ob.eventsMutex.Lock()
	defer ob.eventsMutex.Unlock()

	// Create a copy of the events slice
	eventsCopy := make([]string, len(ob.events))
	copy(eventsCopy, ob.events)
	log.Println("Events:", eventsCopy)

	return eventsCopy
}
