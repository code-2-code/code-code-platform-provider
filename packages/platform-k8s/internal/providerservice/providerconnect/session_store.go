package providerconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	sessionConfigMapLabel   = "platform.code-code.internal/provider-connect-session"
	sessionConfigMapDataKey = "session.json"
)

type sessionStore struct {
	client    ctrlclient.Client
	reader    ctrlclient.Reader
	namespace string
}

func newSessionStore(client ctrlclient.Client, reader ctrlclient.Reader, namespace string) (*sessionStore, error) {
	if client == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: session store client is nil")
	}
	if reader == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: session store reader is nil")
	}
	if strings.TrimSpace(namespace) == "" {
		return nil, fmt.Errorf("platformk8s/providerconnect: session store namespace is empty")
	}
	return &sessionStore{
		client:    client,
		reader:    reader,
		namespace: strings.TrimSpace(namespace),
	}, nil
}

func (s *sessionStore) create(ctx context.Context, record *sessionRecord) error {
	if record == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: session record is nil")
	}
	if strings.TrimSpace(record.SessionID) == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: session id is empty")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("platformk8s/providerconnect: marshal session %q: %w", record.SessionID, err)
	}
	object := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      record.SessionID,
			Namespace: s.namespace,
			Labels: map[string]string{
				sessionConfigMapLabel: "true",
			},
		},
		Data: map[string]string{
			sessionConfigMapDataKey: string(payload),
		},
	}
	if err := s.client.Create(ctx, object); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return domainerror.NewAlreadyExists("platformk8s/providerconnect: session %q already exists", record.SessionID)
		}
		return fmt.Errorf("platformk8s/providerconnect: create session %q: %w", record.SessionID, err)
	}
	return nil
}

func (s *sessionStore) get(ctx context.Context, sessionID string) (*sessionRecord, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: session id is empty")
	}
	object := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: s.namespace, Name: sessionID}
	if err := s.reader.Get(ctx, key, object); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, domainerror.NewNotFound("platformk8s/providerconnect: session %q not found", sessionID)
		}
		return nil, fmt.Errorf("platformk8s/providerconnect: get session %q: %w", sessionID, err)
	}
	payload := strings.TrimSpace(object.Data[sessionConfigMapDataKey])
	if payload == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: session %q payload is empty", sessionID)
	}
	record := &sessionRecord{}
	if err := json.Unmarshal([]byte(payload), record); err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: decode session %q: %w", sessionID, err)
	}
	return record, nil
}

func (s *sessionStore) put(ctx context.Context, record *sessionRecord) error {
	if record == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: session record is nil")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("platformk8s/providerconnect: marshal session %q: %w", record.SessionID, err)
	}
	key := types.NamespacedName{Namespace: s.namespace, Name: record.SessionID}
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		object := &corev1.ConfigMap{}
		if err := s.reader.Get(ctx, key, object); err != nil {
			if apierrors.IsNotFound(err) {
				return domainerror.NewNotFound("platformk8s/providerconnect: session %q not found", record.SessionID)
			}
			return fmt.Errorf("platformk8s/providerconnect: get session %q for update: %w", record.SessionID, err)
		}
		writeSessionRecordPayload(object, string(payload))
		return s.client.Update(ctx, object)
	}); err != nil {
		return fmt.Errorf("platformk8s/providerconnect: update session %q: %w", record.SessionID, err)
	}
	return nil
}

func writeSessionRecordPayload(object *corev1.ConfigMap, payload string) {
	if object.Labels == nil {
		object.Labels = map[string]string{}
	}
	object.Labels[sessionConfigMapLabel] = "true"
	if object.Data == nil {
		object.Data = map[string]string{}
	}
	object.Data[sessionConfigMapDataKey] = payload
}
