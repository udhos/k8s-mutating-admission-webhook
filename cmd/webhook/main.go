// Package main implements the tool.
package main

import (
	"crypto/tls"
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

	addr := envString("ADDR", ":8443")
	route := envString("ROUTE", "/mutate")
	health := envString("HEALTH", "/health")

	//
	// Generate certificate
	//

	const webhookNamespace = "webhook"
	const webhookServiceName = "k8s-mutating-admission-webhook"
	const webhookConfigName = "udhos.github.io"

	dnsNames := []string{
		webhookServiceName,
		webhookServiceName + "." + webhookNamespace,
		webhookServiceName + "." + webhookNamespace + ".svc",
	}
	commonName := webhookServiceName + "." + webhookNamespace + ".svc"

	const org = "github.com/udhos/k8s-mutating-admission-webhook"
	caPEM, certPEM, certKeyPEM, errCert := generateCert([]string{org}, dnsNames, commonName)
	if errCert != nil {
		log.Fatalf("Failed to generate ca and certificate key pair: %v", errCert)
	}

	pair, errPair := tls.X509KeyPair(certPEM.Bytes(), certKeyPEM.Bytes())
	if errPair != nil {
		log.Fatalf("Failed to load certificate key pair: %v", errPair)
	}

	//
	// Add certificate to webhook configuration
	//
	errWebhookConf := createOrUpdateMutatingWebhookConfiguration(caPEM, webhookConfigName, route, webhookServiceName, webhookNamespace)
	if errWebhookConf != nil {
		log.Fatalf("Failed to create or update the mutating webhook configuration: %v", errWebhookConf)
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	const root = "/"

	register(mux, addr, root, func(w http.ResponseWriter, r *http.Request) { handlerRoot(&app, w, r) })
	register(mux, addr, health, func(w http.ResponseWriter, r *http.Request) { handlerHealth(&app, w, r) })
	register(mux, addr, route, func(w http.ResponseWriter, r *http.Request) { handlerRoute(&app, w, r) })

	go func() {
		log.Printf("listening TLS on port %s", addr)
		err := server.ListenAndServeTLS("", "")
		log.Fatalf("listening TLS on port %s: %v", addr, err)
	}()

	<-chan struct{}(nil)
}

func register(mux *http.ServeMux, addr, path string, handler http.HandlerFunc) {
	mux.HandleFunc(path, handler)
	log.Printf("registered on TLS port %s: path %s", addr, path)
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
