// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	api_runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const version = "0.0.0"

func getVersion(me string) string {
	return fmt.Sprintf("%s version=%s runtime=%s GOOS=%s GOARCH=%s GOMAXPROCS=%d",
		me, version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
}

type config struct {
	codecs serializer.CodecFactory
}

func main() {

	app := config{
		codecs: serializer.NewCodecFactory(api_runtime.NewScheme()),
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	{
		v := getVersion(me)
		if showVersion {
			fmt.Println(v)
			return
		}
		log.Print(v)
	}

	addr := envString("ADDR", ":443")
	route := envString("ROUTE", "/mutate")
	health := envString("HEALTH", "/health")
	certFile := envString("TLS_CERT_FILE", "cert.pem")
	keyFile := envString("TLS_KEY_FILE", "key.pem")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	const root = "/"

	register(mux, addr, root, func(w http.ResponseWriter, r *http.Request) { handlerRoot(&app, w, r) })
	register(mux, addr, health, func(w http.ResponseWriter, r *http.Request) { handlerHealth(&app, w, r) })
	register(mux, addr, route, func(w http.ResponseWriter, r *http.Request) { handlerRoute(&app, w, r) })

	go listenAndServeTLS(server, addr, certFile, keyFile)

	<-chan struct{}(nil)
}

func register(mux *http.ServeMux, addr, path string, handler http.HandlerFunc) {
	mux.HandleFunc(path, handler)
	log.Printf("registered on TLS port %s: path %s", addr, path)
}

func listenAndServeTLS(s *http.Server, addr, certFile, keyFile string) {
	log.Printf("listening TLS on port %s", addr)
	err := s.ListenAndServeTLS(certFile, keyFile)
	log.Fatalf("listening TLS on port %s: %v", addr, err)
}

func handlerRoot( /*app*/ _ *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerRoot"
	log.Printf("%s: %s %s %s - 404 not found",
		me, r.RemoteAddr, r.Method, r.RequestURI)
	http.Error(w, "not found", 404)
}

func handlerHealth( /*app*/ _ *config, w http.ResponseWriter, r *http.Request) {
	const me = "handlerHealth"
	log.Printf("%s: %s %s %s - 200 health ok",
		me, r.RemoteAddr, r.Method, r.RequestURI)
	fmt.Fprintln(w, "health ok")
}

// envString extracts string from env var.
// It returns the provided defaultValue if the env var is empty.
// The string returned is also recorded in logs.
func envString(name string, defaultValue string) string {
	str := os.Getenv(name)
	if str != "" {
		log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, str, defaultValue)
		return str
	}
	log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
