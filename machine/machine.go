package machine

import (
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kube-deploy/cluster-api/client"
	"k8s.io/kube-deploy/cluster-api/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubicorn/kubicorn/pkg/logger"
	"fmt"
	"github.com/kubicorn/controller/backoff"
)

func ConcurrentReconcileMachines(cfg *ServiceConfiguration) chan error {
	ch := make(chan error)
	mm := cfg.CloudProvider
	t := backoff.NewBackoff("crm")
	go func() {
		for {
			t.Hang()
			cm, err := getClientMeta(cfg)
			if err != nil {
				ch <- fmt.Errorf("Unable to authenticate client: %v", err)
				continue
			}
			listOptions := metav1.ListOptions{}
			machines, err := cm.client.Machines().List(listOptions)
			if err != nil {
				ch <- fmt.Errorf("Unable to list machines: %v", err)
				continue
			}
			for _, machine := range machines.Items {
				possibleMachine, err := mm.Get(machine.Name)
				if err != nil {
					ch <- fmt.Errorf("Unable to get machine [%s]: %v", machine.Name, err)
					continue
				}
				if possibleMachine == nil {
					// Machine does not exist, create it
					err := mm.Create(&machine)
					if err != nil {
						ch <- fmt.Errorf("Unable to create machine [%s]: %v", machine.Name, err)
						continue
					}
					logger.Debug("Created machine: %s", machine.Name)
					continue
				}
				logger.Debug("Machine already exists: %s", machine.Name)
			}
		}
	}()
	return ch
}

type crdClientMeta struct {
	client    *client.ClusterAPIV1Alpha1Client
	clientset *apiextensionsclient.Clientset
}

func getClientMeta(cfg *ServiceConfiguration) (*crdClientMeta, error) {
	kubeConfigPath, err := cfg.GetFilePath()
	if err != nil {
		return nil, err
	}
	client, err := util.NewApiClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	cs, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	clientMeta := &crdClientMeta{
		client:    client,
		clientset: cs,
	}
	return clientMeta, nil
}
