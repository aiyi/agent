package rsu

import (
	"gopkg.in/mgo.v2"
	"log"
	"time"
)

type Tsdb struct {
	session  *mgo.Session
	obuevent *mgo.Collection
}

func NewTsdb() *Tsdb {
	return &Tsdb{}
}

type eventDoc struct {
	Timestamp     time.Time
	Station       uint16
	Roadway       uint8
	VehicleNumber string
	ObuMAC        string
	VehicleType   uint8
	UserType      uint8
	Tags          []string
}

func (db *Tsdb) WriteObuEvent(event *ObuEvent) error {
	doc := &eventDoc{
		Timestamp:     time.Unix(event.Timestamp, 0),
		Station:       event.Station,
		Roadway:       event.Roadway,
		VehicleNumber: event.VehicleNumber,
		ObuMAC:        event.ObuMAC,
		VehicleType:   event.VehicleType,
		UserType:      event.UserType,
		Tags:          []string{"tag1", "tag2"}}

	err := db.obuevent.Insert(doc)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func (db *Tsdb) Init() error {
	session, err := mgo.Dial("localhost")
	if err == nil {
		db.session = session
		c := session.DB("etc").C("obuevent")
		db.obuevent = c

		index := mgo.Index{
			Key:         []string{"Timestamp"},
			Unique:      false,
			Background:  true,
			Sparse:      true,
			ExpireAfter: 60 * 60 * 24 * 7,
		}
		c.EnsureIndex(index)
		c.EnsureIndexKey("Station")
		c.EnsureIndexKey("Roadway")
		c.EnsureIndexKey("VehicleNumber")
		c.EnsureIndexKey("Tags")
	}
	return err
}

func (db *Tsdb) Exit() {
	if db.session != nil {
		db.session.Close()
	}
}
