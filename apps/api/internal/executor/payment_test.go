package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opspilot/opspilot/apps/api/internal/release"
)

type fakeMutator struct {
	image   string
	version string
	set     []string
	ready   bool
}

func (f *fakeMutator) CurrentPayment(context.Context) (string, string, error) {
	return f.image, f.version, nil
}

func (f *fakeMutator) SetPaymentRelease(_ context.Context, image, version string) error {
	f.set = append(f.set, image+" "+version)
	f.image = image
	f.version = version
	return nil
}

func (f *fakeMutator) WaitPaymentReady(context.Context, string, time.Duration) error {
	if !f.ready && f.image != release.GoodImage {
		return errors.New("not ready")
	}
	return nil
}

func TestPaymentRollbackRefusesOtherTargets(t *testing.T) {
	mut := &fakeMutator{image: release.BadImage, version: release.BadVersion, ready: true}
	err := (PaymentRollback{Mutator: mut}).Execute(context.Background(), Request{
		Action: "rollback", Service: "payment-api", Namespace: "demo-shop", From: "v1.8.2", To: "v1.8.1",
	})
	if err == nil || len(mut.set) != 0 {
		t.Fatalf("err=%v set=%v", err, mut.set)
	}
}

func TestPaymentRollbackOnlyMovesTheBadImageToTheGoodImage(t *testing.T) {
	mut := &fakeMutator{image: release.BadImage, version: release.BadVersion, ready: true}
	err := (PaymentRollback{Mutator: mut}).Execute(context.Background(), Request{
		Action: release.Action, Service: release.Deployment, Namespace: release.Namespace, From: release.BadVersion, To: release.GoodVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mut.set) != 1 || mut.set[0] != release.GoodImage+" "+release.GoodVersion {
		t.Fatalf("%v", mut.set)
	}
	err = (PaymentRollback{Mutator: mut}).Execute(context.Background(), Request{
		Action: release.Action, Service: release.Deployment, Namespace: release.Namespace, From: release.BadVersion, To: release.GoodVersion,
	})
	if err != nil || len(mut.set) != 1 {
		t.Fatalf("second call err=%v set=%v", err, mut.set)
	}
}

func TestPaymentRollbackRefusesAnUnexpectedLiveImage(t *testing.T) {
	mut := &fakeMutator{image: "opspilot-demo:0.3.0", version: "1.4.2"}
	err := (PaymentRollback{Mutator: mut}).Execute(context.Background(), Request{
		Action: release.Action, Service: release.Deployment, Namespace: release.Namespace, From: release.BadVersion, To: release.GoodVersion,
	})
	if err == nil || len(mut.set) != 0 {
		t.Fatalf("err=%v set=%v", err, mut.set)
	}
}
