package provider_test

import (
	"testing"

	"github.com/fwartner/terraform-provider-lockwave/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"lockwave": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestProvider_Instantiation(t *testing.T) {
	p := provider.New("test")
	if p == nil {
		t.Fatal("expected non-nil provider factory")
	}
	instance := p()
	if instance == nil {
		t.Fatal("expected non-nil provider instance")
	}
}

func TestProviderFactories_NotNil(t *testing.T) {
	for name, factory := range testAccProtoV6ProviderFactories {
		if factory == nil {
			t.Errorf("factory for %q is nil", name)
		}
		srv, err := factory()
		if err != nil {
			t.Errorf("factory for %q returned error: %v", name, err)
		}
		if srv == nil {
			t.Errorf("factory for %q returned nil server", name)
		}
	}
}
