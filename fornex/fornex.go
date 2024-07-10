package fornex

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	klog "k8s.io/klog/v2"

	"github.com/090809/cert-manager-webhook-fornex/pkg/fornex"
)

type Solver struct {
	kube *kubernetes.Clientset
}

func (s *Solver) Name() string {
	return "fornex"
}

type Config struct {
	ApiKeySecretRef corev1.SecretKeySelector `json:"apiKeySecretRef"`
}

func (s *Solver) readConfig(request *acme.ChallengeRequest) (*fornex.Client, error) {
	config := Config{}

	if request.Config != nil {
		if err := json.Unmarshal(request.Config.Raw, &config); err != nil {
			return nil, errors.Wrap(err, "config error")
		}
	}

	apiKey, err := s.resolveSecretRef(config.ApiKeySecretRef, request)
	if err != nil {
		return nil, err
	}

	return fornex.New(apiKey), nil
}

func (s *Solver) resolveSecretRef(selector corev1.SecretKeySelector, ch *acme.ChallengeRequest) (string, error) {
	secret, err := s.kube.CoreV1().Secrets(ch.ResourceNamespace).Get(context.Background(), selector.Name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "get error for secret %q %q", ch.ResourceNamespace, selector.Name)
	}

	bytes, ok := secret.Data[selector.Key]
	if !ok {
		return "", errors.Errorf("secret %q %q does not contain key %q", ch.ResourceNamespace, selector.Name, selector.Key)
	}

	return string(bytes), nil
}

func (s *Solver) Present(ch *acme.ChallengeRequest) error {
	klog.Infof("Handling present request for %q %q %q",
		ch.ResolvedFQDN,
		ch.ResolvedZone,
		ch.Key,
	)

	client, err := s.readConfig(ch)
	if err != nil {
		return errors.Wrap(err, "initialization error")
	}

	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	entity := strings.TrimSuffix(ch.ResolvedFQDN, "."+ch.ResolvedZone)
	name := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	records, err := client.RetrieveRecords(context.Background(), domain)
	if err != nil {
		return errors.Wrap(err, "retrieve records error")
	}

	for _, record := range records {
		if record.Type == "TXT" && record.Host == name && record.Value == ch.Key {
			klog.Infof("Record %s is already present", record.ID)
			return nil
		}
	}

	id, err := client.CreateRecord(context.Background(), domain, fornex.Record{
		Host:  entity,
		Type:  "TXT",
		Value: ch.Key,
		TTL:   120,
	})
	if err != nil {
		return errors.Wrap(err, "create record error")
	}

	klog.Infof("Created record %v", id)
	return nil
}

func (s *Solver) CleanUp(ch *acme.ChallengeRequest) error {
	klog.Infof(
		"Handling cleanup request for %q %q %q",
		ch.ResolvedFQDN,
		ch.ResolvedZone,
		ch.Key,
	)

	client, err := s.readConfig(ch)
	if err != nil {
		return errors.Wrap(err, "initialization error")
	}

	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	name := strings.TrimSuffix(ch.ResolvedFQDN, ".")
	records, err := client.RetrieveRecords(context.Background(), domain)
	if err != nil {
		return errors.Wrap(err, "retrieve records error")
	}

	for _, record := range records {
		if record.Type == "TXT" && record.Host == name && record.Value == ch.Key {
			id := record.ID

			record.Value = ch.Key
			err = client.DeleteRecord(context.Background(), domain, id)
			if err != nil {
				return errors.Wrap(err, "delete record error")
			}

			klog.Infof("Deleted record %v", id)
			return nil
		}
	}

	klog.Info("No matching record to delete")

	return nil
}

func (s *Solver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	klog.Info("Initializing")

	kube, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return errors.Wrap(err, "kube client creation error")
	}

	s.kube = kube
	return nil
}

func New() webhook.Solver {
	return &Solver{}
}
