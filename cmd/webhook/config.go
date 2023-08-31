package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type config struct {
	debug                 bool
	addr                  string
	route                 string
	health                string
	namespace             string
	service               string
	webhookConfigName     string
	namespaceExcludeLabel string

	// Ignore: means that an error calling the webhook is ignored and the API request is allowed to continue.
	// Fail: means that an error calling the webhook causes the admission to fail and the API request to be rejected.
	failurePolicy string

	ignoreNamespaces  []string
	removeTolerations []string
}

func getConfig() config {
	return config{
		debug:                 envBool("DEBUG", false),
		addr:                  envString("ADDR", ":8443"),
		route:                 envString("ROUTE", "/mutate"),
		health:                envString("HEALTH", "/health"),
		namespace:             envString("NAMESPACE", "webhook"),
		service:               envString("SERVICE", "k8s-mutating-admission-webhook"),
		webhookConfigName:     envString("WEBHOOK_CONFIG_NAME", "udhos.github.io"),
		namespaceExcludeLabel: envString("NAMESPACE_EXCLUDE_LABEL", "webhook"),
		failurePolicy:         envString("FAILURE_POLICY", "Ignore"),

		// space-separated list of namespaces
		ignoreNamespaces: strings.Fields(envString("IGNORE_NAMESPACES", "karpenter")),

		// space-separated list of tolerations
		removeTolerations: strings.Fields(envString("REMOVE_TOLERATIONS", "CriticalAddonsOnly")),
	}
}

// envString extracts string from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envString(name string, defaultValue string) string {
	str := os.Getenv(name)
	if str != "" {
		log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, str, defaultValue)
		return str
	}
	log.Printf("%s=[%s] using %s=%s default=%s", name, str, name, defaultValue, defaultValue)
	return defaultValue
}

// envBool extracts bool from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envBool(name string, defaultValue bool) bool {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := strconv.ParseBool(str)
		if errConv == nil {
			log.Printf("%s=[%s] using %s=%t default=%t", name, str, name, value, defaultValue)
			return value
		}
		log.Printf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	log.Printf("%s=[%s] using %s=%t default=%t", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
