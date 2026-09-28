package release

import "testing"

func TestProposeOnlyAllowsTheKnownBadRelease(t *testing.T) {
	bad := Propose(BadVersion)
	if !bad.Allowed || bad.Action != Action || bad.From != BadVersion || bad.To != GoodVersion {
		t.Fatalf("%#v", bad)
	}
	for _, version := range []string{"", "1.4.2", "0.3.0", "v1.8.2", GoodImage} {
		item := Propose(version)
		if item.Allowed || item.Action != "stop-experiment" {
			t.Fatalf("version %s proposed %#v", version, item)
		}
	}
}

func TestValidateRejectsAnythingExceptTheFixedRollback(t *testing.T) {
	if err := Validate(Action, Deployment, Namespace, BadVersion, GoodVersion); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ action, service, namespace, from, to string }{
		{"rollback", Deployment, Namespace, BadVersion, GoodVersion},
		{Action, "storefront", Namespace, BadVersion, GoodVersion},
		{Action, Deployment, "opspilot-system", BadVersion, GoodVersion},
		{Action, Deployment, Namespace, GoodVersion, BadVersion},
		{Action, Deployment, Namespace, BadVersion, "latest"},
	}
	for _, tc := range cases {
		if err := Validate(tc.action, tc.service, tc.namespace, tc.from, tc.to); err == nil {
			t.Fatalf("accepted %#v", tc)
		}
	}
}

func TestKnownImagesAreAClosedPair(t *testing.T) {
	if !Known(GoodImage, GoodVersion) || !Known(BadImage, BadVersion) {
		t.Fatal("known pair missing")
	}
	if Known(BadImage, GoodVersion) || Known("nginx:latest", BadVersion) || Known(GoodImage, "1.4.3") {
		t.Fatal("unknown pair accepted")
	}
}
