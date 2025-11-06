package main

import (
	"bytes"
	"context"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func certAutocheck(clientset *kubernetes.Clientset,
	expectedCaPEM []byte, webhookConfigName string,
	interval time.Duration, maxErrors int) {

	const me = "certAutocheck"

	var errors int

	for errors < maxErrors {
		time.Sleep(interval)

		mutatingWebhookConfigV1Client := clientset.AdmissionregistrationV1()

		foundWebhookConfig, err := mutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Get(context.TODO(), webhookConfigName, metav1.GetOptions{})

		if err != nil {
			errors++
			log.Printf("%s: ERROR %d of %d: retrieving webhook=%s config: %v",
				me, errors, maxErrors, webhookConfigName, err)
			continue
		}

		if len(foundWebhookConfig.Webhooks) != 1 {
			errors++
			log.Printf("%s: ERROR %d of %d: wrong number of webhook=%s configs: found=%d expected=1",
				me, errors, maxErrors, webhookConfigName, len(foundWebhookConfig.Webhooks))
			continue
		}

		wh := foundWebhookConfig.Webhooks[0]

		if wh.Name != webhookConfigName {
			errors++
			log.Printf("%s: ERROR %d of %d: wrong webhook=%s name: %s",
				me, errors, maxErrors, webhookConfigName, wh.Name)
			continue
		}

		caBundle := wh.ClientConfig.CABundle

		if !bytes.Equal(caBundle, expectedCaPEM) {
			errors++
			log.Printf("%s: ERROR %d of %d: wrong webhook=%s certificate: expected=%s got=%s",
				me, errors, maxErrors, webhookConfigName, expectedCaPEM, caBundle)
			continue
		}

		log.Printf("%s: webhook=%s: ok", me, webhookConfigName)

		errors = 0 // reset errors
	}

	log.Fatalf("%s: webhook=%s: FATAL: reached error limit %d of %d",
		me, webhookConfigName, errors, maxErrors)
}
