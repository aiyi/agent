package rsu

import (
	. "github.com/aiyi/agent/agent"
	rest "github.com/emicklei/go-restful"
	"net/http"
)

type Heartbeat struct {
	Interval int
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
