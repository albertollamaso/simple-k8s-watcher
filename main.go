package main

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	extensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const Annotation = "external-dns.alpha.kubernetes.io/hostname"

func failOnError(err error, msg string) {
	if err != nil {
		log.Error("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))

	}
}

func getClient(pathToCfg string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if pathToCfg == "" {
		log.Info("Using in cluster config")
		config, err = rest.InClusterConfig()
		// in cluster access
	} else {
		log.Info("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", pathToCfg)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func logginglevel() {
	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to debug
	if !ok {
		logLevel = "info"
	}
	// parse string, this is built-in feature of logrus
	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.DebugLevel
	}

	// set global log level
	log.SetLevel(ll)

}

func iterateIngresses(ingress *extensionsV1beta1.IngressList) {
	var ingressFound = false
	var lenAnnotations = 0
	if len(ingress.Items) > 0 {
		ingressFound = true
		for _, ingress := range ingress.Items {
			if len(ingress.Annotations[Annotation]) > 0 {
				lenAnnotations++
				log.Info(ingress.Annotations[Annotation])
			}
		}

	}

	if ingressFound == false {
		log.Info("None Ingress found in the cluster")
	}

	if lenAnnotations == 0 {
		log.Info("Not found at least one ingress with annotation: %s", Annotation)
	}

}

func main() {

	// CONFIGURE LOGGING LEVEL
	logginglevel()

	// AUTHENTICATE
	clientset, err := getClient("config.yaml")
	failOnError(err, "Failed to create kubernetes clientset")

	// GET SERVICES RESOURCE VERSION
	log.Info("Getting Services resources version")
	var api = clientset.ExtensionsV1beta1().Ingresses("")
	ingresses, err := api.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	log.Debug("Ingresses:\n%s", ingresses)
	resourceVersion := ingresses.ListMeta.ResourceVersion
	log.Debug("Resource Version: ", resourceVersion)

	// GETTING THE INITIAL STATE OF TE WORLD
	log.Info("Getting current state of the world. Show all annotations that already exists in the cluster")
	iterateIngresses(ingresses)

	// SETUP WATCHER CHANNEL
	log.Info("Setting up a watcher channel")
	watcher, err := api.Watch(context.TODO(), metav1.ListOptions{ResourceVersion: resourceVersion})
	if err != nil {
		panic(err.Error())
	}
	ch := watcher.ResultChan()

	// LISTEN TO CHANNEL
	log.Info("Watching for changes...")
	for {
		event := <-ch
		ingresses, ok := event.Object.(*extensionsV1beta1.Ingress)
		if !ok {
			panic("Could not cast to Endpoint")
		}
		log.WithFields(log.Fields{
			"Domain": ingresses.Annotations[Annotation],
			"Event":  event.Type,
		}).Info("A change has been detected")
	}
}
