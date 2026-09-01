package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sethvargo/go-retry"

	"modelmesh/pkg/log"

	"modelmesh/pkg/core"
	"modelmesh/pkg/mesh"
)

const maxBody = 8 << 20 // 1 MiB

const MeshModelPrefix = ""

var proxyHandleURLS = []string{
	// ollama
	"/api/chat",
	"/api/embed",
	"/api/generate",
	// open ai
	"/v1/chat/completions",
	"/v1/responses",
	"/v1/embeddings",
	// Anthropic
	"/v1/messages",
}

type RequestPeek struct {
	Model string
}

type Proxy struct {
	listen string
	cid    uint64

	mux     *http.ServeMux
	meshMux *http.ServeMux // handler for requests coming in via the mesh if we want something different
	mesh    *mesh.Service

	lock       sync.RWMutex
	meshRoutes map[string]*ModelRoute // contains all models discovered in the mesh
	//meshRoutesOAI map[string]*ModelRoute // contains all models discovered in the mesh
	localRoutes map[string]*ModelRoute // contain the mapping of a model to a ollama server configured to this node
}

func (p *Proxy) getLocalModelRoute(model string) string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	route, ok := p.localRoutes[model]
	if !ok {
		return ""
	}

	for k := range route.Servers {
		return k
	}
	return ""
}

func (p *Proxy) getMeshModelRoute(model string) string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	router, ok := p.meshRoutes[model]
	if !ok {
		return ""
	}

	for peerId := range router.Servers {
		if peerId == p.mesh.ID {
			continue
		}
		return peerId
	}
	return ""
}

func (p *Proxy) peekModel(body []byte) string {
	peek := RequestPeek{}
	if err := json.Unmarshal(body, &peek); err != nil {
		return ""
	}
	// trim of the MeshModelPrefix from model names to indicate the model is coming from the mesh pool
	return strings.TrimPrefix(strings.ToLower(peek.Model), MeshModelPrefix)
}

func (p *Proxy) proxyModelRequest(w http.ResponseWriter, r *http.Request, noRelay bool) {
	limited := io.LimitReader(r.Body, maxBody+1)
	body, err := io.ReadAll(limited)
	r.Body.Close()

	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	model := p.peekModel(body)
	log.Debugf(" -- Model: %s\n", model)

	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	// Always try local routes first
	route := p.getLocalModelRoute(model)
	if route != "" {
		u, err := url.Parse(route)
		if err != nil {
			http.Error(w, "model not found", http.StatusNotFound)
			return
		}
		log.Debugf(" -- Servicing via ollama node: %s\n", route)

		proxy := httputil.NewSingleHostReverseProxy(u)
		orig := proxy.Director
		proxy.Director = func(req *http.Request) {
			orig(req)
			req.Host = u.Host
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		}
		proxy.ServeHTTP(w, r)
		return
	}

	// if no relay is set, this request came in via the mesh we should not send it back to the mesh as
	// we could loop forever
	if noRelay {
		log.Debugf(" -- No relay set, returning 404")
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	route = p.getMeshModelRoute(model)
	if route == "" {
		log.Debugf(" -- No model available, returning 404")
		http.Error(w, "no model available", http.StatusNotFound)
		return
	}

	log.Debugf(" -- Servicing via mesh node: %s\n", route)
	p.mesh.ProxyToNode(route, w, r)
	return
}

// OnPeerUpdate handles the addition or removal of a peer and updates the meshRoutes accordingly.
// It fetches model information from peers on addition and merges it into the local model state.
// Returns an error if the model fetch from a peer fails.
func (p *Proxy) OnPeerUpdate(peerID string, remove bool) error {
	log.Eventf("ollama.proxy.OnPeerUpdate:%s [Remove:%t]\n", peerID, remove)
	// fetch models from peer
	if remove {
		p.lock.Lock()
		for _, model := range p.meshRoutes {
			_, ok := model.Servers[peerID]
			if ok {
				log.Debugf("Peer Remove Model: %s/%s\n", peerID, model.Name)
			}
		}
		p.lock.Unlock()
		return nil
	}

	// Peers that registered at the same time are often not dialable yet
	// (circuit reservation / swarm backoff). Retry before giving up;
	// discovery will try again on the next poll if this still fails.
	client := NewClient(peerID, p.mesh.ClientForPeer(peerID))

	var models map[string]*ModelRoute
	err := retry.Do(context.Background(), retry.WithMaxRetries(3, retry.NewFibonacci(2*time.Second)),
		func(ctx context.Context) error {
			var err error
			models, err = client.GetModelsMesh()
			if err != nil {
				log.Warnf("error fetching models from peer %s: %s; retrying", peerID, err)
				return retry.RetryableError(err)
			}
			return nil
		})

	if err != nil {
		log.Errorf("error fetching models from peer: %s", err)
		return err
	}

	// merge models into out model state
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, model := range models {
		log.Debugf(" -- ollama model: %s/%s\n", peerID, model.Name)
		route, ok := p.meshRoutes[model.Name]
		if !ok {
			route = &ModelRoute{
				Name:       model.Name,
				Model:      model.Model,
				Process:    model.Process,    // these should be synthesized
				Properties: model.Properties, // these should be synthesized
				Servers:    make(map[string]string),
			}
			p.meshRoutes[model.Name] = route
		}
		route.Servers[peerID] = peerID
	}
	return nil
}

// ServeHTTP serves an Ollama compatible api for chat completions
//
//	this handler is exposed to all local clients that want to use for access
//	to local and remote models.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cid := atomic.AddUint64(&p.cid, 1)
	start := time.Now()

	log.WithName("proxy").Eventf("%s -- (local:%d) %s %s\n", r.RemoteAddr, cid, r.Method, r.URL.Path)
	switch r.URL.Path {
	case "/api/chat", "/v1/chat/completions", "/v1/responses":
		p.proxyModelRequest(w, r, false)
	default: // serves from our local table
		p.mux.ServeHTTP(w, r)
	}
	log.WithName("proxy").Eventf("%s %v (local:%d) %s %s\n", r.RemoteAddr, time.Now().Sub(start).Round(time.Second), cid, r.Method, r.URL.Path)
}

// MeshServeHTTP serves only our local models, it is used as an entry point for our p2p peers when they ask for
//
//	us to answer for them as an edge node
func (p *Proxy) MeshServeHTTP(w http.ResponseWriter, r *http.Request) {
	cid := atomic.AddUint64(&p.cid, 1)
	start := time.Now()

	log.WithName("proxy").Eventf("%s -- (mesh:%d) %s %s\n", r.RemoteAddr, cid, r.Method, r.URL.Path)
	switch {
	case slices.Contains(proxyHandleURLS, r.URL.Path): // pivots on model
		p.proxyModelRequest(w, r, true)
	default: // serves from our local table
		p.mux.ServeHTTP(w, r)
	}
	log.WithName("proxy").Eventf("%s %v (mesh:%d) %s %s\n", r.RemoteAddr, time.Now().Sub(start).Round(time.Second), cid, r.Method, r.URL.Path)
}

func (p *Proxy) Serve(ctx context.Context) error {
	p.mesh.Start()

	svr := http.Server{
		Handler: p,
		Addr:    p.listen,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 600 * time.Second,
		ReadTimeout:  600 * time.Second,
		TLSConfig:    &tls.Config{},
	}
	go func() {
		err := svr.ListenAndServe()
		if err != nil {
			log.WithName("proxy").Eventf("proxy Service Failed: %s", err)
		}
	}()

	log.WithName("proxy").Eventf("Proxy Service Started")
	<-ctx.Done()
	return svr.Shutdown(context.Background())
}

// NewProxy creates a local proxy that routes ollama requests based on model name to a specific
// endpoint on the network
func NewProxy(meshService *mesh.Service, listen string, providers []core.Provider) (*Proxy, error) {
	p := &Proxy{
		listen:      listen,
		mux:         http.NewServeMux(),
		meshRoutes:  make(map[string]*ModelRoute),
		mesh:        meshService,
		localRoutes: make(map[string]*ModelRoute),
	}

	// Loads providers
	exports := NewModelExporter(providers)
	exports.Refresh()
	models := exports.List()

	// initialize our local models
	for idx := range models {
		localModel := models[idx]

		p.localRoutes[localModel.Model] = &localModel

		meshModel := models[idx]
		meshModel.Servers = map[string]string{meshService.ID: meshService.ID}
		p.meshRoutes[meshModel.Model] = &meshModel
	}

	// OLLAMA Specific APIs
	p.mux.HandleFunc("GET /api/ps", p.apiListProcessHandler)
	p.mux.HandleFunc("GET /api/tags", p.apiListTagsHandler)

	// OpenAI APIs
	p.mux.HandleFunc("GET /v1/models", p.openaiListModelsHandler)

	// Notes to AI: .mesh endpoints are only to be used by PEER to PEER requests.  Fo UI the /api/mesh/ endpoints
	p.mux.HandleFunc("GET /.mesh/status", p.apiMeshStatus)
	p.mux.HandleFunc("GET /.mesh/members", p.apiMeshMembers)
	p.mux.HandleFunc("GET /.mesh/models", p.apiMeshModels)

	// /api/mesh/... are the api endpoints that can be used by UIs/clients
	p.mux.HandleFunc("GET /api/mesh/models", p.apiUIModelsHandler)
	p.mux.HandleFunc("GET /api/mesh/members", p.apiMeshMembers)
	p.mux.HandleFunc("GET /api/mesh/config", p.apiUIConfigHandler)

	p.mux.HandleFunc("GET /{$}", p.uiRootHandler)
	p.mux.HandleFunc("GET /ui", p.uiHandler)
	p.mux.Handle("GET /ui/", http.StripPrefix("/ui/", http.FileServer(uiFileSystem())))

	p.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("404: The page you are looking for does not exist. %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Custom 404: The page you are looking for does not exist.")
	})

	p.mesh.Handler(p.MeshServeHTTP)
	p.mesh.Discovery().UpdateHandler(p.OnPeerUpdate)

	return p, nil
}
