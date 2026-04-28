package providercatalogs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const catalogProbeLeaseDurationSeconds int32 = 300

type catalogProbeLeaseHeldError struct {
	name string
}

func (e catalogProbeLeaseHeldError) Error() string {
	return fmt.Sprintf("platformk8s/providercatalogs: model catalog probe %q is already running", e.name)
}

func (e *CatalogProbeExecutor) acquireCatalogProbeLease(ctx context.Context, key string) (func(), error) {
	name := catalogProbeLeaseName(key)
	holder := catalogProbeLeaseHolder()
	now := metav1.NewMicroTime(time.Now().UTC())
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: e.leaseNamespace,
			Name:      name,
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       stringPtr(holder),
			LeaseDurationSeconds: int32Ptr(catalogProbeLeaseDurationSeconds),
			AcquireTime:          &now,
			RenewTime:            &now,
		},
	}
	if err := e.client.Create(ctx, lease); err == nil {
		return e.catalogProbeLeaseRelease(name, holder), nil
	} else if !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current := &coordinationv1.Lease{}
		if err := e.client.Get(ctx, types.NamespacedName{Namespace: e.leaseNamespace, Name: name}, current); err != nil {
			return err
		}
		if catalogProbeLeaseHeldByOther(current, holder, time.Now().UTC()) {
			return catalogProbeLeaseHeldError{name: name}
		}
		now := metav1.NewMicroTime(time.Now().UTC())
		current.Spec.HolderIdentity = stringPtr(holder)
		current.Spec.LeaseDurationSeconds = int32Ptr(catalogProbeLeaseDurationSeconds)
		current.Spec.RenewTime = &now
		if current.Spec.AcquireTime == nil || catalogProbeLeaseExpired(current, now.Time) {
			current.Spec.AcquireTime = &now
		}
		return e.client.Update(ctx, current)
	}); err != nil {
		return nil, err
	}
	return e.catalogProbeLeaseRelease(name, holder), nil
}

func (e *CatalogProbeExecutor) catalogProbeLeaseRelease(name string, holder string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			current := &coordinationv1.Lease{}
			if err := e.client.Get(ctx, types.NamespacedName{Namespace: e.leaseNamespace, Name: name}, current); err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			if current.Spec.HolderIdentity == nil || *current.Spec.HolderIdentity != holder {
				return nil
			}
			return e.client.Delete(ctx, current)
		})
	}
}

func catalogProbeLeaseHeldByOther(lease *coordinationv1.Lease, holder string, now time.Time) bool {
	if lease == nil || lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity == holder {
		return false
	}
	return !catalogProbeLeaseExpired(lease, now)
}

func catalogProbeLeaseExpired(lease *coordinationv1.Lease, now time.Time) bool {
	if lease == nil || lease.Spec.RenewTime == nil {
		return true
	}
	duration := time.Duration(catalogProbeLeaseDurationSeconds) * time.Second
	if lease.Spec.LeaseDurationSeconds != nil && *lease.Spec.LeaseDurationSeconds > 0 {
		duration = time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second
	}
	return lease.Spec.RenewTime.Time.Add(duration).Before(now)
}

func catalogProbeLeaseName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "model-catalog-probe-" + hex.EncodeToString(sum[:])[:40]
}

func catalogProbeLeaseHolder() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

func stringPtr(value string) *string {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}
