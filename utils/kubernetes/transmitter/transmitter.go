package transmitter

import (
	"context"
	"fmt"

	logger "github.com/totvs/go-sdk/log"
	"github.com/totvs/go-sdk/utils/pipeline"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// crdTransmitter is a generic Transmitter that creates/updates CRDs on a target Kubernetes cluster.
// The concrete CRD type is erased via closure in NewCRDTransmitter[T].
type crdTransmitter struct {
	transmitFn func(ctx context.Context, report pipeline.Report) error
}

// NewCRDTransmitter creates a Transmitter decoupled from the concrete CRD type.
// Uses function-level generics (same pattern as pipeline.NewStep[T,R]).
func NewCRDTransmitter[T client.Object](
	k8sClient client.Client,
	clusterName string,
	resourceName string,
	mapper CRDMapper[T],
) pipeline.Transmitter {
	if k8sClient == nil {
		panic("k8sClient cannot be nil")
	}
	if clusterName == "" {
		panic("clusterName cannot be empty")
	}
	if mapper == nil {
		panic("mapper cannot be nil")
	}

	t := &crdTransmitter{}

	t.transmitFn = func(ctx context.Context, report pipeline.Report) error {
		l := logger.FromContext(ctx)
		l.Debug().Msgf("[CRDTransmitter] Transmitting report for cluster '%s'", clusterName)

		key := client.ObjectKey{Name: resourceName, Namespace: clusterName}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			obj := mapper.NewObject()

			err := k8sClient.Get(ctx, key, obj)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("failed to get CRD resource: %w", err)
				}

				obj = mapper.MapToCreate(clusterName)
				obj.SetName(resourceName)
				obj.SetNamespace(clusterName)
				if err := k8sClient.Create(ctx, obj); err != nil {
					l.Error(err).Msg("[CRDTransmitter] Failed to create resource")
					return fmt.Errorf("failed to create CRD resource: %w", err)
				}
				l.Debug().Msg("[CRDTransmitter] Resource created successfully")

				if err := k8sClient.Get(ctx, key, obj); err != nil {
					return fmt.Errorf("failed to re-fetch CRD resource after create: %w", err)
				}
			}

			mapper.MapToStatus(obj, report)

			if err := k8sClient.Status().Update(ctx, obj); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			l.Error(err).Msg("[CRDTransmitter] Failed to update status after retries")
			return fmt.Errorf("failed to update CRD status: %w", err)
		}

		l.Debug().Msg("[CRDTransmitter] Status updated successfully")
		return nil
	}

	return t
}

func (t *crdTransmitter) Transmit(ctx context.Context, report pipeline.Report) error {
	return t.transmitFn(ctx, report)
}
