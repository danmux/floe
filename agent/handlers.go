package agent

import (
	"fmt"
	"net/http"
	"time"

	"github.com/floeit/floe/log"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

type execInstruction struct {
	Command string
	Delay   time.Duration
}

// get all floes from all the agents
func allFloeHandler(w http.ResponseWriter, req *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", thisAgent.project.RenderableProject()
}

// return the summary and run summaries for a specific floe
func floeHandler(w http.ResponseWriter, req *http.Request, ctx *context) (int, string, renderable) {
	// confirm floe id given in params
	flID := ctx.ps.ByName("flid")
	if flID == "" {
		return rNotFound, "floe id not specified", nil
	}

	// TODO - query all other agents - and add agent to the RenderableFloe
	return rOK, "", thisAgent.project.RenderableFloe(flID)
}

// return the status of a current run
func runHandler(w http.ResponseWriter, req *http.Request, ctx *context) (int, string, renderable) {
	// confirm floe id given in params
	flID := ctx.ps.ByName("flid")
	if flID == "" {
		return rNotFound, "floe id not specified", nil
	}

	agentID := ctx.ps.ByName("agentid")
	if flID == "" {
		return rNotFound, "run id not specified", nil
	}

	runID := ctx.ps.ByName("runid")
	if flID == "" {
		return rNotFound, "run id not specified", nil
	}

	println(agentID, runID)

	return rOK, "", nil
}

// execute the floe given in the url
func execHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {

	// confirm floe id given in params
	flID := ctx.ps.ByName("flid")
	if flID == "" {
		return rNotFound, "floe Id not specified", nil
	}

	v := execInstruction{
		Delay: 1,
	}
	if ok := decodeBody(rw, r, &v); !ok {
		return 0, "", nil
	}

	v.Delay = v.Delay * time.Second

	rid, err := ctx.agent.execAsync(flID, v.Delay)
	if err != nil {
		return rErr, err.Error(), nil
	}

	rw.Header()["Location"] = []string{fmt.Sprintf("%s/floes/%s/run/%s/%d", rootPath, flID, ctx.agent.ref.ID, rid)}
	// which agent was it run on
	return rCreated, "", struct {
		FloeID  string
		AgentID string
		RunID   int
	}{flID, "a1", rid}
}

func stopHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	// confirm floe id given in parms
	flID := ctx.ps.ByName("flid")
	if flID == "" {
		return rNotFound, "floe Id not specified", nil
	}

	err := ctx.agent.stop(flID)
	if err != nil {
		return rErr, err.Error(), nil
	}

	return rOK, "stopped", nil
}

func loginHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	v := struct {
		User     string
		Password string
	}{}
	if ok := decodeBody(rw, r, &v); !ok {
		return 0, "", nil
	}

	token := login(v.User, v.Password)
	if token == "" {
		return rUnauth, "username or password were wrong", nil
	}

	setCookie(rw, token)

	// authenticated if we got here
	return rOK, "", struct {
		User  string
		Role  string
		Token string
	}{v.User, "ADMIN", token}
}

func logoutHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	if ctx.sesh == nil {
		return rBad, "token not supplied", nil
	}
	logout(ctx.sesh.token)
	return rOK, "logged out", nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // ignore Origin header
}

func wsHandler(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Info("new inbound ws connection")
	// --- do the ws magic
	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Error("upgrade error", err)
		return
	}

	// now we have upgraded the connection to a web socket - launch the message handler
	handleWS(ws)
}

func indexHandler(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	rw.Write([]byte(`
<!doctype html>
<html lang="en" data-framework="marionettejs">
	<head>
		<meta charset="utf-8">
		<title>Floe</title>

        <link href='//fonts.googleapis.com/css?family=Open+Sans:300italic,400italic,600italic,400,300,600' rel='stylesheet' type='text/css'>
        <link href="//maxcdn.bootstrapcdn.com/font-awesome/4.2.0/css/font-awesome.min.css" rel="stylesheet">
		<link rel="stylesheet" href="css/bootstrap.min.css">
        <link rel="stylesheet" href="css/dagre.css">
        <link rel="stylesheet" href="css/highlight/default.css">
        <script src="js/lib/highlight.pack.js"></script>

        <link rel="stylesheet" href="css/style.css">

        <link rel=icon href="img/floe-small.png" sizes="32x32" type="image/png">

        <!-- HTML5 shim and Respond.js IE8 support of HTML5 elements and media queries -->
        <!--[if lt IE 9]>
        <script src="https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js"></script>
        <script src="https://oss.maxcdn.com/libs/respond.js/1.4.2/respond.min.js"></script>
        <![endif]-->
	</head>

    <body>
        <div class="container">
            <div class="header">
                <span id="main-nav">
                </span>
                <h3 class="text-muted"><span><img src="img/floe-big.png" height="32"></span>loe</h3>
            </div>

            <div id="notification"></div><!-- /#notification -->

            <div id="main">
            </div>

            <div class="footer" id="footer">
            </div>

            <!-- To be used as modal and notifications-->
            <div class="modal fade" id="dialog" tabindex="-1" role="dialog" aria-hidden="true">
            </div><!-- /.modal -->

        </div> <!-- /container -->

        <script src="js/lib/d3.js"></script>
        <script src="js/lib/dagre-d3.js"></script>
        <script data-main="js/app/main" src="js/lib/requirejs/require.js"></script>
    </body>

</html>

		`))
}
