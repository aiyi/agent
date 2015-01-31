package rsu

import (
	. "github.com/aiyi/agent/agent"
	iconv "github.com/djimenez/iconv-go"
	rest "github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"log"
	"net/http"
)

var Iconv *iconv.Converter

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

func (u RsuService) Register() {
	ws := new(rest.WebService)
	ws.Path("/RSU").
		Doc("查询和设置RSU工作参数").
		Consumes(rest.MIME_JSON).
		Produces(rest.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.GET("/Online").To(u.findOnlineRsu).
		Doc("查询在线RSU").
		Operation("findOnlineRsu").
		Returns(200, "OK", []RsuInfo{}))
	ws.Route(ws.POST("/{IP}/OpenAnt").To(u.openAnt).
		Doc("打开RSU天线").
		Operation("openAnt").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Returns(200, "OK", nil))
	ws.Route(ws.POST("/{IP}/CloseAnt").To(u.closeAnt).
		Doc("关闭RSU天线").
		Operation("closeAnt").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Returns(200, "OK", nil))
	ws.Route(ws.GET("/{IP}/StaRoad").To(u.getStaRoad).
		Doc("查询RSU站点和车道").
		Operation("getStaRoad").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(StaRoad{}))
	ws.Route(ws.GET("/{IP}/Channel").To(u.getChannel).
		Doc("查询RSU通信信道号").
		Operation("getChannel").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(Channel{}))
	ws.Route(ws.GET("/{IP}/TxPower").To(u.getTxPower).
		Doc("查询RSU发射功率级数").
		Operation("getTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(TxPower{}))
	ws.Route(ws.GET("/{IP}/RevSensitive").To(u.getRevSensitive).
		Doc("查询RSU接收灵敏度").
		Operation("getRevSensitive").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(RevSensitive{}))
	ws.Route(ws.PUT("/{IP}/StaRoad").To(u.setStaRoad).
		Doc("设置RSU站点和车道").
		Operation("setStaRoad").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(StaRoad{}))
	ws.Route(ws.PUT("/{IP}/TxPower").To(u.setTxPower).
		Doc("设置RSU发射功率级数").
		Operation("setTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(TxPower{}))
	ws.Route(ws.PUT("/{IP}/RevSensitive").To(u.setRevSensitive).
		Doc("设置RSU接收灵敏度").
		Operation("setRevSensitive").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(RevSensitive{}))

	rest.Add(ws)
}

func (u RsuService) findOnlineRsu(request *rest.Request, response *rest.Response) {
	a := u.agentd
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

func (u RsuService) getClient(request *rest.Request, response *rest.Response) (*Conn, *RsuProtoInst, bool) {
	a := u.agentd
	ip := request.PathParameter("IP")

	c, ok := a.GetClient(ip)
	if !ok {
		response.WriteError(http.StatusNotFound, RsuNotFoundError)
		return nil, nil, false
	}

	p := c.ProtoInstance().(*RsuProtoInst)
	return c, p, true
}

func (u RsuService) openAnt(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) closeAnt(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) getStaRoad(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) getChannel(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) getTxPower(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) setTxPower(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) getRevSensitive(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) setStaRoad(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

func (u RsuService) setRevSensitive(request *rest.Request, response *rest.Response) {
	c, p, ok := u.getClient(request, response)
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

type RestServer struct {
	agentd *AgentD
}

func NewRestServer(a *AgentD) *RestServer {
	r := &RestServer{
		agentd: a,
	}
	return r
}

func restServer(r *RestServer) {
	log.Printf("start listening on %s:8080", r.agentd.GetServerIP())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (r *RestServer) Main() {
	u := RsuService{}
	u.agentd = r.agentd
	u.Register()

	url := "http://" + r.agentd.GetServerIP() + ":8080"

	// Optionally, you can install the Swagger Service which provides a nice Web UI on your REST API
	// You need to download the Swagger HTML5 assets and change the FilePath location in the config below.
	// Open http://localhost:8080/apidocs and enter http://localhost:8080/apidocs.json in the api input field.
	config := swagger.Config{
		WebServices:    rest.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl: url,
		ApiPath:        "/apidocs.json",

		// Optionally, specifiy where the UI is located
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: "/root/go/src/github.com/wordnik/swagger-ui/dist"}
	swagger.InstallSwaggerService(config)

	go restServer(r)

	Iconv, _ = iconv.NewConverter("gb2312", "utf-8")
}

func (r *RestServer) Exit() {

}
