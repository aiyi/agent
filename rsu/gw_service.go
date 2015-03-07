package rsu

import (
	. "github.com/aiyi/agent/agent"
	rest "github.com/emicklei/go-restful"
	"net/http"
	"strconv"
)

type Heartbeat struct {
	Interval int
}

type Tags struct {
	Tags []string
}

type Target struct {
	ObuMAC string
}

type GwService struct {
	agentd *AgentD
}

func (s GwService) Register() {
	ws := new(rest.WebService)
	ws.Path("/GW").
		Doc("网关系统功能接口").
		Consumes(rest.MIME_JSON).
		Produces(rest.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.GET("/OnlineRSU").To(s.findOnlineRsu).
		Doc("查询在线RSU").
		Operation("findOnlineRsu").
		Returns(200, "OK", []RsuInfo{}))

	ws.Route(ws.GET("/OBUEvent").To(s.getObuEvent).
		Doc("查询OBU事件").
		Operation("getObuEvent").
		Param(ws.QueryParameter("FromDate", "开始时间(2006-01-02 15:04:05)").DataType("string")).
		Param(ws.QueryParameter("ToDate", "结束时间(2006-01-02 15:04:05)").DataType("string")).
		Param(ws.QueryParameter("Station", "站点号").DataType("integer")).
		Param(ws.QueryParameter("Roadway", "车道号").DataType("integer")).
		Param(ws.QueryParameter("VehicleNumber", "车牌号码").DataType("string")).
		Param(ws.QueryParameter("Tags", "标签(tag1,tag2)").DataType("string")).
		Returns(200, "OK", []EventDoc{}))

	ws.Route(ws.PUT("/Heartbeat").To(s.setHeartbeatInterval).
		Doc("设置心跳消息间隔").
		Operation("setHeartbeatInterval").
		Reads(Heartbeat{}))

	ws.Route(ws.GET("/Tags").To(s.listTags).
		Doc("查询站点/车道标签").
		Operation("listTags").
		Returns(200, "OK", []TagDoc{}))

	ws.Route(ws.PUT("/{Station}/{Roadway}/Tags").To(s.setTags).
		Doc("设置站点/车道标签").
		Operation("setTags").
		Param(ws.PathParameter("Station", "站点号").DataType("integer")).
		Param(ws.PathParameter("Roadway", "车道号").DataType("integer")).
		Reads(Tags{}))

	ws.Route(ws.GET("/Targets").To(s.listTargets).
		Doc("查询锁定目标OBU").
		Operation("listTargets").
		Returns(200, "OK", []TargetDoc{}))

	ws.Route(ws.PUT("/Target").To(s.addTarget).
		Doc("增加锁定目标OBU").
		Operation("addTarget").
		Reads(Target{}))

	ws.Route(ws.DELETE("/Target").To(s.deleteTarget).
		Doc("删除锁定目标OBU").
		Operation("deleteTarget").
		Reads(Target{}))

	rest.Add(ws)
}

func (s GwService) findOnlineRsu(request *rest.Request, response *rest.Response) {
	a := s.agentd
	rsus := []*RsuInfo{}

	a.RLock()
	for _, c := range a.Clients {
		rsu := &RsuInfo{
			IP: c.String(),
		}
		rsus = append(rsus, rsu)
	}
	a.RUnlock()

	response.WriteEntity(rsus)
}

func (s GwService) getObuEvent(request *rest.Request, response *rest.Response) {
	from := request.QueryParameter("FromDate")
	to := request.QueryParameter("ToDate")
	station := request.QueryParameter("Station")
	roadway := request.QueryParameter("Roadway")
	vehicle := request.QueryParameter("VehicleNumber")
	tags := request.QueryParameter("Tags")

	events := &[]EventDoc{}
	db.FindObuEvent(from, to, station, roadway, vehicle, tags, events)

	response.WriteEntity(events)
}

func (s GwService) setHeartbeatInterval(request *rest.Request, response *rest.Response) {
	ent := new(Heartbeat)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	if ent.Interval <= 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	HBInterval = uint32(ent.Interval)
	response.WriteEntity(ent)
}

func (s GwService) listTags(request *rest.Request, response *rest.Response) {
	err, tagDocs := db.ListTag()
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}
	response.WriteEntity(tagDocs)
}

func (s GwService) setTags(request *rest.Request, response *rest.Response) {
	var sta uint16
	var rd uint8
	station := request.PathParameter("Station")
	roadway := request.PathParameter("Roadway")

	if station != "" {
		i, err := strconv.ParseUint(station, 10, 16)
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		sta = uint16(i)
	}
	if roadway != "" {
		i, err := strconv.ParseUint(roadway, 10, 8)
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		rd = uint8(i)
	}

	ent := new(Tags)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	err = db.UpdateTag(sta, rd, ent.Tags)
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	response.WriteEntity(ent)
}

func (s GwService) listTargets(request *rest.Request, response *rest.Response) {
	err, targetDocs := db.ListTarget()
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}
	response.WriteEntity(targetDocs)
}

func (s GwService) addTarget(request *rest.Request, response *rest.Response) {
	ent := new(Target)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	err = db.AddTarget(ent.ObuMAC)
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	response.WriteEntity(ent)
}

func (s GwService) deleteTarget(request *rest.Request, response *rest.Response) {
	ent := new(Target)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	err = db.DeleteTarget(ent.ObuMAC)
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	response.WriteHeader(http.StatusOK)
}
