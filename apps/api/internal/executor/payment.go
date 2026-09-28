package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

// PaymentMutator is the only cluster write the payment rollback may use.
type PaymentMutator interface {
	CurrentPayment(ctx context.Context) (image, version string, err error)
	SetPaymentRelease(ctx context.Context, image, version string) error
	WaitPaymentReady(ctx context.Context, image string, timeout time.Duration) error
}

// PaymentRollback rolls demo-shop/payment-api from the known bad image to the known good image.
type PaymentRollback struct {
	Mutator PaymentMutator
}

func (p PaymentRollback) Execute(ctx context.Context, req Request) error {
	if p.Mutator == nil {
		return fmt.Errorf("%w: payment rollback is not configured", release.ErrDenied)
	}
	if err := release.Validate(req.Action, req.Service, req.Namespace, req.From, req.To); err != nil {
		return err
	}
	image, version, err := p.Mutator.CurrentPayment(ctx)
	if err != nil {
		return err
	}
	if image == release.GoodImage && version == release.GoodVersion {
		return nil
	}
	if image != release.BadImage || version != release.BadVersion {
		return fmt.Errorf("%w: payment-api is %s (%s)", release.ErrDenied, image, version)
	}
	if err := p.Mutator.SetPaymentRelease(ctx, release.GoodImage, release.GoodVersion); err != nil {
		return err
	}
	return p.Mutator.WaitPaymentReady(ctx, release.GoodImage, 90*time.Second)
}
