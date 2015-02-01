package rsu

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strconv"
	"strings"
	"time"
)

type Tsdb struct {
	session  *mgo.Session
	obuevent *mgo.Collection
}

func NewTsdb() (*Tsdb, error) {
	db := &Tsdb{}

	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}

	db.session = session
	c := session.DB("etc").C("obuevent")
	db.obuevent = c

	index := mgo.Index{
		Key:         []string{"DateTime"},
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

	return db, nil
}

type EventDoc struct {
	DateTime      time.Time
	Station       uint16
	Roadway       uint8
	VehicleNumber string
	ObuMAC        string
	VehicleType   uint8
	UserType      uint8
	Tags          []string
}

func (d *Tsdb) WriteObuEvent(event *ObuEvent) error {
	doc := &EventDoc{
		DateTime:      time.Unix(event.Timestamp, 0),
		Station:       event.Station,
		Roadway:       event.Roadway,
		VehicleNumber: event.VehicleNumber,
		ObuMAC:        event.ObuMAC,
		VehicleType:   event.VehicleType,
		UserType:      event.UserType,
		Tags:          []string{"tag1", "tag2"}}

	err := d.obuevent.Insert(doc)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func (d *Tsdb) FindObuEvent(from, to, station, roadway, vehicle, tags string, events *[]EventDoc) {
	queryM := bson.M{}
	periodM := bson.M{}

	if from != "" {
		fromDate, err := time.ParseInLocation("2006-01-02 15:04:05", from, time.Local)
		if err != nil {
			return
		}
		periodM["$gt"] = fromDate
	}
	if to != "" {
		toDate, err := time.ParseInLocation("2006-01-02 15:04:05", to, time.Local)
		if err != nil {
			return
		}
		periodM["$lt"] = toDate
	}
	if len(periodM) > 0 {
		queryM["datetime"] = periodM
	}
	if station != "" {
		i, err := strconv.ParseUint(station, 10, 16)
		if err != nil {
			return
		}
		queryM["station"] = uint16(i)
	}
	if roadway != "" {
		i, err := strconv.ParseUint(roadway, 10, 8)
		if err != nil {
			return
		}
		queryM["roadway"] = uint8(i)
	}
	if vehicle != "" {
		queryM["vehiclenumber"] = vehicle
	}
	if tags != "" {
		taglist := strings.Split(tags, ",")
		if len(taglist) > 0 {
			tagM := bson.M{"$all": taglist}
			queryM["tags"] = tagM
		}
	}

	fmt.Println(queryM)

	err := d.obuevent.Find(queryM).Limit(100).All(events)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Tsdb) Close() {
	if d.session != nil {
		d.session.Close()
	}
}
