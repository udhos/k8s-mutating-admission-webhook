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

	_ "github.com/KimMachineGun/automemlimit"
	"github.com/udhos/kube/kubeclient"
	"gopkg.in/yaml.v3"
	api_runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func getVersion(me string) string {
	return fmt.Sprintf("%s version=%s runtime=%s GOOS=%s GOARCH=%s GOMAXPROCS=%d",
		me, version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
}

type application struct {
	codecs serializer.CodecFactory
	conf   config
	rules  rulesList
}

func main() {

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

	app := application{
		codecs: serializer.NewCodecFactory(api_runtime.NewScheme()),
		conf:   getConfig(),
	}

	{
		r, errRules := loadRules(app.conf.rulesFile)
		if errRules != nil {
			log.Fatalf("rules load: %s: %v", app.conf.rulesFile, errRules)
		}

		out, errY := yaml.Marshal(r)
		if errY != nil {
			log.Fatalf("rules yaml: %v", errY)
		}
		log.Printf("rules loaded:\n%s", string(out))

		app.rules = r
	}

	//
	// Generate certificate
	//

	webhookNamespace := app.conf.namespace
	webhookServiceName := app.conf.service
	webhookConfigName := app.conf.webhookConfigName

	dnsNames := []string{
		webhookServiceName,
		webhookServiceName + "." + webhookNamespace,
		webhookServiceName + "." + webhookNamespace + ".svc",
	}
	commonName := webhookServiceName + "." + webhookNamespace + ".svc"

	const org = "github.com/udhos/k8s-mutating-admission-webhook"
	caPEM, certPEM, certKeyPEM, errCert := generateCert([]string{org},
		dnsNames, commonName, app.conf.certDurationYears)
	if errCert != nil {
		log.Fatalf("Failed to generate ca and certificate key pair: %v", errCert)
	}

	pair, errPair := tls.X509KeyPair(certPEM, certKeyPEM)
	if errPair != nil {
		log.Fatalf("Failed to load certificate key pair: %v", errPair)
	}

	//
	// Create kube client
	//
	options := kubeclient.Options{
		DebugLog: true,
	}
	clientset, errClient := kubeclient.New(options)
	if errClient != nil {
		log.Fatalf("Failed to create kube client: %v", errClient)
	}

	//
	// Add certificate to webhook configuration
	//

	errWebhookConf := createOrUpdateMutatingWebhookConfiguration(clientset,
		caPEM, webhookConfigName, app.conf.route, webhookServiceName,
		webhookNamespace, app.conf.failurePolicy,
		app.conf.namespaceExcludeLabel, app.conf.reinvocationPolicy)
	if errWebhookConf != nil {
		log.Fatalf("Failed to create or update the mutating webhook configuration: %v", errWebhookConf)
	}

	//
	// Spawn certificate auto-check
	//

	if app.conf.certAutocheck {
		go certAutocheck(clientset, caPEM, webhookConfigName,
			app.conf.certAutocheckInterval, app.conf.certAutocheckErrorLimit)
	}

	//
	// Create web server
	//

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:      app.conf.addr,
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	//
	// Register routes
	//

	const root = "/"

	register(mux, app.conf.addr, root, func(w http.ResponseWriter, r *http.Request) { handlerRoot(&app, w, r) })
	register(mux, app.conf.addr, app.conf.health, func(w http.ResponseWriter, r *http.Request) { handlerHealth(&app, w, r) })
	register(mux, app.conf.addr, app.conf.route, func(w http.ResponseWriter, r *http.Request) { handlerWebhook(&app, w, r) })

	//
	// Start web server
	//

	log.Printf("listening TLS on port %s", app.conf.addr)
	err := server.ListenAndServeTLS("", "")
	log.Fatalf("listening TLS on port %s: %v", app.conf.addr, err)
}

func register(mux *http.ServeMux, addr, path string, handler http.HandlerFunc) {
	mux.HandleFunc(path, handler)
	log.Printf("registered on TLS port %s: path %s", addr, path)
}

func handlerRoot( /*app*/ _ *application, w http.ResponseWriter, r *http.Request) {
	const me = "handlerRoot"
	log.Printf("%s: %s %s %s - 404 not found",
		me, r.RemoteAddr, r.Method, r.RequestURI)
	http.Error(w, "not found", 404)
}

func handlerHealth( /*app*/ _ *application, w http.ResponseWriter, _ /*r*/ *http.Request) {
	fmt.Fprintln(w, "health ok")
}
