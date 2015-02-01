package rsu

import (
	. "github.com/aiyi/agent/agent"
	rest "github.com/emicklei/go-restful"
	"net/http"
)

type RsuInfo struct {
	IP string
}

type StaRoad struct {
	Station int
	Roadway uint8
}

type Channel struct {
	Channel uint8
}

type TxPower struct {
	TxPower uint8
}

type RevSensitive struct {
	RevSensitive uint8
}

type RsuService struct {
	agentd *AgentD
}

func (s RsuService) Register() {
	ws := new(rest.WebService)
	ws.Path("/RSU").
		Doc("查询和设置RSU工作参数").
		Consumes(rest.MIME_JSON).
		Produces(rest.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.POST("/{IP}/OpenAnt").To(s.openAnt).
		Doc("打开RSU天线").
		Operation("openAnt").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Returns(200, "OK", nil))
	ws.Route(ws.POST("/{IP}/CloseAnt").To(s.closeAnt).
		Doc("关闭RSU天线").
		Operation("closeAnt").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Returns(200, "OK", nil))
	ws.Route(ws.GET("/{IP}/StaRoad").To(s.getStaRoad).
		Doc("查询RSU站点和车道").
		Operation("getStaRoad").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(StaRoad{}))
	ws.Route(ws.GET("/{IP}/Channel").To(s.getChannel).
		Doc("查询RSU通信信道号").
		Operation("getChannel").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(Channel{}))
	ws.Route(ws.GET("/{IP}/TxPower").To(s.getTxPower).
		Doc("查询RSU发射功率级数").
		Operation("getTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(TxPower{}))
	ws.Route(ws.GET("/{IP}/RevSensitive").To(s.getRevSensitive).
		Doc("查询RSU接收灵敏度").
		Operation("getRevSensitive").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(RevSensitive{}))
	ws.Route(ws.PUT("/{IP}/StaRoad").To(s.setStaRoad).
		Doc("设置RSU站点和车道").
		Operation("setStaRoad").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(StaRoad{}))
	ws.Route(ws.PUT("/{IP}/TxPower").To(s.setTxPower).
		Doc("设置RSU发射功率级数").
		Operation("setTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(TxPower{}))
	ws.Route(ws.PUT("/{IP}/RevSensitive").To(s.setRevSensitive).
		Doc("设置RSU接收灵敏度").
		Operation("setRevSensitive").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(RevSensitive{}))

	rest.Add(ws)
}

func (s RsuService) getClient(request *rest.Request, response *rest.Response) (*Conn, *RsuProtoInst, bool) {
	a := s.agentd
	ip := request.PathParameter("IP")

	c, ok := a.GetClient(ip)
	if !ok {
		response.WriteError(http.StatusNotFound, RsuNotFoundError)
		return nil, nil, false
	}

	p := c.ProtoInstance().(*RsuProtoInst)
	return c, p, true
}

func (s RsuService) openAnt(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, e := c.SendCommand(p.NewOpenAntMsg())
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).GetRsuStatus() != 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	response.WriteEntity(nil)
}

func (s RsuService) closeAnt(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, e := c.SendCommand(p.NewCloseAntMsg())
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).GetRsuStatus() != 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	response.WriteEntity(nil)
}

func (s RsuService) getStaRoad(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, err := c.SendCommand(p.NewGetStaRoadMsg())
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	ent := new(StaRoad)
	ent.Station = int(resp.(*RsuMessage).GetStation())
	ent.Roadway = resp.(*RsuMessage).GetRoadway()
	response.WriteEntity(ent)
}

func (s RsuService) getChannel(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, err := c.SendCommand(p.NewGetChannelMsg())
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	ent := new(Channel)
	ent.Channel = resp.(*RsuMessage).GetChannel()
	response.WriteEntity(ent)
}

func (s RsuService) getTxPower(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, err := c.SendCommand(p.NewGetTxPowerMsg())
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	ent := new(TxPower)
	ent.TxPower = resp.(*RsuMessage).GetTxPower()
	response.WriteEntity(ent)
}

func (s RsuService) setTxPower(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	ent := new(TxPower)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, e := c.SendCommand(p.NewSetTxPowerMsg(ent.TxPower))
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).GetRsuStatus() != 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	response.WriteEntity(ent)
}

func (s RsuService) getRevSensitive(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	resp, err := c.SendCommand(p.NewGetRevSensitiveMsg())
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	ent := new(RevSensitive)
	ent.RevSensitive = resp.(*RsuMessage).GetRevSensitive()
	response.WriteEntity(ent)
}

func (s RsuService) setStaRoad(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	ent := new(StaRoad)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, e := c.SendCommand(p.NewSetStaRoadMsg(uint16(ent.Station), ent.Roadway))
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).GetRsuStatus() != 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	response.WriteEntity(ent)
}

func (s RsuService) setRevSensitive(request *rest.Request, response *rest.Response) {
	c, p, ok := s.getClient(request, response)
	if !ok {
		return
	}

	ent := new(RevSensitive)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, e := c.SendCommand(p.NewSetRevSensitiveMsg(ent.RevSensitive))
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).GetRsuStatus() != 0 {
		response.WriteError(http.StatusExpectationFailed, SetParameterError)
		return
	}

	response.WriteEntity(ent)
}
