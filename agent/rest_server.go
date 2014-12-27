package agent

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"log"
	"net/http"
)

type Rsu struct {
	Ip, Name string
}

type RsuService struct {
	agentd *AgentD
}

func (u RsuService) Register() {
	ws := new(restful.WebService)
	ws.
		Path("/rsu").
		Doc("Manage RSU").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON) // you can specify this per route as well

	ws.Route(ws.GET("/").To(u.findOnlineRsu).
		// docs
		Doc("get all connected RSU").
		Operation("findOnlineRsu").
		Returns(200, "OK", []Rsu{}))

	ws.Route(ws.GET("/{IP}").To(u.queryPara).
		// docs
		Doc("get parameters of RSU").
		Operation("queryPara").
		Param(ws.PathParameter("IP", "IP Address").DataType("string")).
		Writes(Rsu{})) // on the response

	restful.Add(ws)
}

func (u RsuService) findOnlineRsu(request *restful.Request, response *restful.Response) {
	a := u.agentd
	rsus := []*Rsu{}

	a.RLock()
	for _, c := range a.clients {
		rsu := &Rsu{
			Ip:   c.addr,
			Name: c.addr,
		}
		rsus = append(rsus, rsu)
	}
	a.RUnlock()

	response.WriteEntity(rsus)
}

func (u RsuService) queryPara(request *restful.Request, response *restful.Response) {
	a := u.agentd
	ip := request.PathParameter("IP")

	c, ok := a.GetClient(ip)
	if !ok {
		response.WriteErrorString(http.StatusNotFound, "RSU could not be found.")
		return
	}

	rsu := new(Rsu)
	rsu.Ip = ip
	rsu.Name = c.addr
	response.WriteEntity(rsu)
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
		WebServices:    restful.RegisteredWebServices(), // you control what services are visible
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
