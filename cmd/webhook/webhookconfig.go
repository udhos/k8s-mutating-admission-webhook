package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func createOrUpdateMutatingWebhookConfiguration(caPEM *bytes.Buffer, webhookConfigName, webhookPath, webhookService, webhookNamespace, failurePolicy string) error {
	log.Println("Initializing the kube client...")

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, errHome := os.UserHomeDir()
		if errHome != nil {
			log.Printf("Could not get home dir: %v", errHome)
		}
		kubeconfig = filepath.Join(home, "/.kube/config")
	}

	config, errKubeconfig := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if errKubeconfig != nil {
		log.Printf("kubeconfig: %v", errKubeconfig)

		c, errInCluster := rest.InClusterConfig()
		if errInCluster != nil {
			log.Printf("in-cluster-config: %v", errInCluster)
		}
		config = c
	}

	if config == nil {
		return errors.New("could not get cluster config")
	}

	clientset, errConfig := kubernetes.NewForConfig(config)
	if errConfig != nil {
		return errConfig
	}

	mutatingWebhookConfigV1Client := clientset.AdmissionregistrationV1()

	log.Printf("Creating or updating the mutatingwebhookconfiguration: %s", webhookConfigName)

	//fail := admissionregistrationv1.Fail
	fp := admissionregistrationv1.FailurePolicyType(failurePolicy)

	sideEffect := admissionregistrationv1.SideEffectClassNone
	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookConfigName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name:                    webhookConfigName,
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			SideEffects:             &sideEffect,
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caPEM.Bytes(), // self-generated CA for the webhook
				Service: &admissionregistrationv1.ServiceReference{
					Name:      webhookService,
					Namespace: webhookNamespace,
					Path:      &webhookPath,
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{
				{
					Operations: []admissionregistrationv1.OperationType{
						admissionregistrationv1.Create,
						admissionregistrationv1.Update,
					},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			NamespaceSelector: &metav1.LabelSelector{
				/*
					MatchLabels: map[string]string{
						"sidecar-injection": "enabled",
					},
				*/

				// exclude namespaces with label webhook=anything
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "webhook",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
				},
			},
			FailurePolicy: &fp,

			// AdmissionWebhookMatchConditions alpha in 1.27
			// https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
			MatchConditions: []admissionregistrationv1.MatchCondition{
				{
					Name:       "excludeWebhook",
					Expression: "!object.metadata.name.matches('^" + webhookService + "-.*$')",
				},
			},
		}},
	}

	foundWebhookConfig, err := mutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Get(context.TODO(), webhookConfigName, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		if _, err := mutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Create(context.TODO(), mutatingWebhookConfig, metav1.CreateOptions{}); err != nil {
			log.Printf("Failed to create the mutatingwebhookconfiguration: %s", webhookConfigName)
			return err
		}
		log.Printf("Created mutatingwebhookconfiguration: %s", webhookConfigName)
	} else if err != nil {
		log.Printf("Failed to check the mutatingwebhookconfiguration: %s", webhookConfigName)
		return err
	} else {
		// there is an existing mutatingWebhookConfiguration
		if len(foundWebhookConfig.Webhooks) != len(mutatingWebhookConfig.Webhooks) ||
			!(foundWebhookConfig.Webhooks[0].Name == mutatingWebhookConfig.Webhooks[0].Name &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].AdmissionReviewVersions, mutatingWebhookConfig.Webhooks[0].AdmissionReviewVersions) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].SideEffects, mutatingWebhookConfig.Webhooks[0].SideEffects) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].FailurePolicy, mutatingWebhookConfig.Webhooks[0].FailurePolicy) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].Rules, mutatingWebhookConfig.Webhooks[0].Rules) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].NamespaceSelector, mutatingWebhookConfig.Webhooks[0].NamespaceSelector) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].ClientConfig.CABundle, mutatingWebhookConfig.Webhooks[0].ClientConfig.CABundle) &&
				reflect.DeepEqual(foundWebhookConfig.Webhooks[0].ClientConfig.Service, mutatingWebhookConfig.Webhooks[0].ClientConfig.Service)) {
			mutatingWebhookConfig.ObjectMeta.ResourceVersion = foundWebhookConfig.ObjectMeta.ResourceVersion
			if _, err := mutatingWebhookConfigV1Client.MutatingWebhookConfigurations().Update(context.TODO(), mutatingWebhookConfig, metav1.UpdateOptions{}); err != nil {
				log.Printf("Failed to update the mutatingwebhookconfiguration: %s", webhookConfigName)
				return err
			}
			log.Printf("Updated the mutatingwebhookconfiguration: %s", webhookConfigName)
		} else {
			log.Printf("The mutatingwebhookconfiguration: %s already exists and has no change", webhookConfigName)
		}
	}

	return nil
}
