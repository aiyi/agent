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
	session   *mgo.Session
	obueventC *mgo.Collection
	tagC      *mgo.Collection
	targetC   *mgo.Collection
	tagM      map[uint32]*TagDoc
	targetM   map[string]*TargetDoc
}

func NewTsdb() (*Tsdb, error) {
	db := &Tsdb{
		tagM:    make(map[uint32]*TagDoc),
		targetM: make(map[string]*TargetDoc),
	}

	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}

	db.session = session
	c := session.DB("etc").C("obuevent")
	db.obueventC = c

	indexes, err := c.Indexes()
	if err != nil {
		return nil, err
	}

	if len(indexes) < 5 {
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
	}

	db.tagC = session.DB("etc").C("tag")
	db.targetC = session.DB("etc").C("target")

	_, tagDocs := db.ListTag()
	for _, tagDoc := range *tagDocs {
		staRd := uint32(tagDoc.Station)<<16 | uint32(tagDoc.Roadway)
		db.tagM[staRd] = &tagDoc
	}

	_, targetDocs := db.ListTarget()
	for _, targetDoc := range *targetDocs {
		db.targetM[targetDoc.ObuMAC] = &targetDoc
	}

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

type TagDoc struct {
	Station uint16
	Roadway uint8
	Tags    []string
}

type TargetDoc struct {
	ObuMAC string
}

func (d *Tsdb) WriteObuEvent(event *ObuEvent) error {
	var tags []string
	staRd := uint32(event.Station)<<16 | uint32(event.Roadway)
	tagDoc, ok := d.tagM[staRd]
	if ok {
		tags = tagDoc.Tags
	}

	doc := &EventDoc{
		DateTime:      time.Unix(event.Timestamp, 0),
		Station:       event.Station,
		Roadway:       event.Roadway,
		VehicleNumber: event.VehicleNumber,
		ObuMAC:        event.ObuMAC,
		VehicleType:   event.VehicleType,
		UserType:      event.UserType,
		Tags:          tags}

	err := d.obueventC.Insert(doc)
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

	err := d.obueventC.Find(queryM).Limit(100).All(events)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Tsdb) ListTag() (error, *[]TagDoc) {
	tagdocs := &[]TagDoc{}
	err := d.tagC.Find(bson.M{}).All(tagdocs)
	if err != nil {
		log.Fatal(err)
		return err, nil
	} else {
		return nil, tagdocs
	}
}

func (d *Tsdb) UpdateTag(station uint16, roadway uint8, tags []string) error {
	doc := &TagDoc{
		Station: station,
		Roadway: roadway,
		Tags:    tags}

	_, err := d.tagC.Upsert(bson.M{"station": station, "roadway": roadway}, doc)
	if err != nil {
		log.Fatal(err)
		return err
	}

	staRd := uint32(station)<<16 | uint32(roadway)
	d.tagM[staRd] = doc
	return nil
}

func (d *Tsdb) ListTarget() (error, *[]TargetDoc) {
	targetdocs := &[]TargetDoc{}
	err := d.targetC.Find(bson.M{}).All(targetdocs)
	if err != nil {
		log.Fatal(err)
		return err, nil
	} else {
		return nil, targetdocs
	}
}

func (d *Tsdb) AddTarget(ObuMAC string) error {
	doc := &TargetDoc{
		ObuMAC: ObuMAC}

	_, err := d.targetC.Upsert(bson.M{"obumac": ObuMAC}, doc)
	if err != nil {
		log.Fatal(err)
		return err
	}

	d.targetM[ObuMAC] = doc
	return nil
}

func (d *Tsdb) DeleteTarget(ObuMAC string) error {
	err := d.targetC.Remove(bson.M{"obumac": ObuMAC})
	if err != nil {
		return err
	}

	delete(d.targetM, ObuMAC)
	return nil
}

func (d *Tsdb) TargetIsLocated(ObuMAC string) bool {
	_, ok := d.targetM[ObuMAC]
	if ok {
		return true
	}
	return false
}

func (d *Tsdb) Close() {
	if d.session != nil {
		d.session.Close()
	}
}
