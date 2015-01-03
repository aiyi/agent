package rsu

import (
	. "github.com/aiyi/agent/agent"
	rest "github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"log"
	"net/http"
)

type RsuInfo struct {
	IP string
}

type TxPower struct {
	TxPower uint8
}

type RsuService struct {
	agentd *AgentD
}

func (u RsuService) Register() {
	ws := new(rest.WebService)
	ws.
		Path("/RSU").
		Doc("查询和设置RSU工作参数").
		Consumes(rest.MIME_JSON).
		Produces(rest.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.GET("/").To(u.findOnlineRsu).
		Doc("查询在线RSU").
		Operation("findOnlineRsu").
		Returns(200, "OK", []RsuInfo{}))
	ws.Route(ws.GET("/{IP}/TxPower").To(u.getTxPower).
		Doc("查询RSU发射功率级数").
		Operation("getTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Writes(TxPower{}))
	ws.Route(ws.PUT("/{IP}").To(u.setTxPower).
		Doc("设置RSU发射功率级数").
		Operation("setTxPower").
		Param(ws.PathParameter("IP", "IP地址").DataType("string")).
		Reads(TxPower{}))
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

func (u RsuService) getTxPower(request *rest.Request, response *rest.Response) {
	a := u.agentd
	ip := request.PathParameter("IP")

	c, ok := a.GetClient(ip)
	if !ok {
		response.WriteError(http.StatusNotFound, RsuNotFoundError)
		return
	}

	proto := c.ProtoInstance().(*RsuProtoInst)
	resp, err := c.SendCommand(proto.NewGetTxPowerMsg())
	if err != nil {
		response.WriteError(http.StatusExpectationFailed, err)
		return
	}

	ent := new(TxPower)
	ent.TxPower = resp.(*RsuMessage).TxPower()
	response.WriteEntity(ent)
}

func (u RsuService) setTxPower(request *rest.Request, response *rest.Response) {
	a := u.agentd
	ip := request.PathParameter("IP")

	ent := new(TxPower)
	err := request.ReadEntity(&ent)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	c, ok := a.GetClient(ip)
	if !ok {
		response.WriteError(http.StatusNotFound, RsuNotFoundError)
		return
	}

	proto := c.ProtoInstance().(*RsuProtoInst)
	resp, e := c.SendCommand(proto.NewSetTxPowerMsg(ent.TxPower))
	if e != nil {
		response.WriteError(http.StatusExpectationFailed, e)
		return
	}

	if resp.(*RsuMessage).RsuStatus() != 0 {
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
}

func (r *RestServer) Exit() {

}
