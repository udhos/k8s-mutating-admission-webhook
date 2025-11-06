package main

import (
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	debug                   bool
	addr                    string
	route                   string
	health                  string
	namespace               string
	service                 string
	webhookConfigName       string
	namespaceExcludeLabel   string
	certDurationYears       int
	certAutocheck           bool
	certAutocheckInterval   time.Duration
	certAutocheckErrorLimit int

	// Ignore: means that an error calling the webhook is ignored and the API request is allowed to continue.
	// Fail: means that an error calling the webhook causes the admission to fail and the API request to be rejected.
	failurePolicy string

	reinvocationPolicy  string
	ignoreNamespaces    []string
	acceptNodeSelectors []string

	rulesFile string
}

func getConfig() config {
	return config{
		debug:                   envBool("DEBUG", false),
		addr:                    envString("ADDR", ":8443"),
		route:                   envString("ROUTE", "/mutate"),
		health:                  envString("HEALTH", "/health"),
		namespace:               envString("NAMESPACE", "webhook"),
		service:                 envString("SERVICE", "k8s-mutating-admission-webhook"),
		webhookConfigName:       envString("WEBHOOK_CONFIG_NAME", "udhos.github.io"),
		namespaceExcludeLabel:   envString("NAMESPACE_EXCLUDE_LABEL", "webhook"),
		certDurationYears:       envInt("CERT_DURATION_YEARS", 10),
		certAutocheck:           envBool("CERT_AUTOCHECK", true),
		certAutocheckInterval:   envDuration("CERT_AUTOCHECK_INTERVAL", 10*time.Second),
		certAutocheckErrorLimit: envInt("CERT_AUTOCHECK_ERROR_LIMIT", 3),
		failurePolicy:           envString("FAILURE_POLICY", "Ignore"),
		reinvocationPolicy:      envString("REINVOCATION_POLICY", "IfNeeded"),

		// space-separated list of namespaces
		ignoreNamespaces: strings.Fields(envString("IGNORE_NAMESPACES", "karpenter")),

		// space-separated list of nodeSelectors
		acceptNodeSelectors: strings.Fields(envString("ACCEPT_NODE_SELECTORS", "kubernetes.io/os")),

		rulesFile: envString("RULES", "rules.yaml"),
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

// envInt extracts int from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envInt(name string, defaultValue int) int {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := strconv.ParseInt(str, 10, 64)
		if errConv == nil {

			// Check for potential overflow/underflow before converting to int
			if value > math.MaxInt || value < math.MinInt {
				log.Printf("WARNING: %s=[%s] value %d is out of range for int. Using default value %d",
					name, str, value, defaultValue)
				return defaultValue
			}

			log.Printf("%s=[%s] using %s=%d default=%d", name, str, name, value, defaultValue)
			return int(value)
		}
		log.Printf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	log.Printf("%s=[%s] using %s=%d default=%d", name, str, name, defaultValue, defaultValue)
	return defaultValue
}

// envDuration extracts Duration from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envDuration(name string, defaultValue time.Duration) time.Duration {
	str := os.Getenv(name)
	if str != "" {
		value, errConv := time.ParseDuration(str)
		if errConv == nil {
			log.Printf("%s=[%s] using %s=%d default=%d", name, str, name, value, defaultValue)
			return value
		}
		log.Printf("bad %s=[%s]: error: %v", name, str, errConv)
	}
	log.Printf("%s=[%s] using %s=%d default=%d", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
