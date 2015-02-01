package rsu

import (
	. "github.com/aiyi/agent/agent"
	"github.com/djimenez/iconv-go"
	rest "github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"log"
	"net/http"
	"os"
)

var (
	db   *Tsdb
	conv *iconv.Converter
)

type RestServer struct {
	agentd *AgentD
}

func NewRestServer(a *AgentD) *RestServer {
	var err error
	db, err = NewTsdb()
	if err != nil {
		log.Fatal("FATAL: failed to connect to database - %s", err)
		os.Exit(1)
	}

	conv, _ = iconv.NewConverter("gb2312", "utf-8")

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
	rsuSvc := &RsuService{r.agentd}
	rsuSvc.Register()

	gwSvc := &GwService{r.agentd}
	gwSvc.Register()

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
	if db != nil {
		db.Close()
	}
}
