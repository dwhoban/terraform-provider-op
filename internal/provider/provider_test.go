// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"op": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "" || desktopAccountName() != "" {
		return
	}
	t.Skip("OP_SERVICE_ACCOUNT_TOKEN or a signed-in `op` account not set; skipping acceptance test")
}

func TestFirstOPAccountUUID(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{name: "first account uuid", data: `[{"account_uuid":"AAA"},{"account_uuid":"BBB"}]`, want: "AAA"},
		{name: "empty list", data: `[]`, want: ""},
		{name: "invalid json", data: `not json`, want: ""},
	}
	for _, tc := range cases {
		if got := firstOPAccountUUID([]byte(tc.data)); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestProviderSchemaDescribesAuthAttributes(t *testing.T) {
	resp := &provider.SchemaResponse{}
	New("test")().Schema(context.Background(), provider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema error: %s", resp.Diagnostics.Errors())
	}
	for _, name := range []string{"service_account_token", "desktop_account_name", "integration_name"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("schema missing auth attribute %q", name)
		}
	}
}
