// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"sync"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &onepasswordProvider{}

// New returns the provider factory consumed by main.go.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &onepasswordProvider{version: version}
	}
}

type onepasswordProvider struct {
	version string
}

type onepasswordProviderModel struct {
	ServiceAccountToken types.String `tfsdk:"service_account_token"`
	DesktopAccountName  types.String `tfsdk:"desktop_account_name"`
	IntegrationName     types.String `tfsdk:"integration_name"`
}

func (p *onepasswordProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "op"
	resp.Version = p.version
}

func (p *onepasswordProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_account_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "1Password service account token. May also be set via the `OP_SERVICE_ACCOUNT_TOKEN` environment variable. Takes precedence over desktop app integration.",
			},
			"desktop_account_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "1Password account name (or UUID) for desktop app integration. May also be set via the `OP_DESKTOP_ACCOUNT_NAME` environment variable. When unset, the local `op` CLI's default account is used.",
			},
			"integration_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Integration name reported to 1Password. Defaults to `terraform-provider-op`.",
			},
		},
	}
}

func (p *onepasswordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config onepasswordProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ServiceAccountToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("service_account_token"),
			"Unknown service account token",
			"service_account_token depends on a value known only after apply. Set a static value or use the OP_SERVICE_ACCOUNT_TOKEN environment variable.",
		)
	}
	if config.DesktopAccountName.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("desktop_account_name"),
			"Unknown desktop account name",
			"desktop_account_name depends on a value known only after apply. Set a static value or use the OP_DESKTOP_ACCOUNT_NAME environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	token := config.ServiceAccountToken.ValueString()
	if token == "" {
		token = os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	}
	desktopAccount := config.DesktopAccountName.ValueString()
	if desktopAccount == "" {
		desktopAccount = os.Getenv("OP_DESKTOP_ACCOUNT_NAME")
	}
	integrationName := config.IntegrationName.ValueString()
	if integrationName == "" {
		integrationName = "terraform-provider-op"
	}

	if token == "" && desktopAccount == "" {
		// Fall back to the local `op` CLI's default account so local
		// execution needs no configuration when the desktop app is signed in.
		desktopAccount = defaultOPAccountName()
	}
	if token == "" && desktopAccount == "" {
		resp.Diagnostics.AddError(
			"Missing 1Password credentials",
			"Set service_account_token in the provider block or export OP_SERVICE_ACCOUNT_TOKEN, set desktop_account_name / OP_DESKTOP_ACCOUNT_NAME for desktop app integration, or sign in to the 1Password desktop app with the `op` CLI installed.",
		)
		return
	}

	var opts []onepassword.ClientOption
	opts = append(opts, onepassword.WithIntegrationInfo(integrationName, p.version))
	if token != "" {
		opts = append(opts, onepassword.WithServiceAccountToken(token))
	} else {
		opts = append(opts, onepassword.WithDesktopAppIntegration(desktopAccount))
	}

	client, err := onepassword.NewClient(ctx, opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create 1Password client",
			err.Error(),
		)
		return
	}

	resp.ResourceData = client
	resp.DataSourceData = client
	resp.EphemeralResourceData = client
	resp.ActionData = client
}

func (p *onepasswordProvider) Resources(_ context.Context) []func() resource.Resource {
	out := []func() resource.Resource{
		NewVaultResource,
		NewItemResource,
	}
	for _, s := range categoryItemSpecs {
		out = append(out, func(s categorySpec) func() resource.Resource {
			return func() resource.Resource { return NewCategoryItemResource(s) }
		}(s))
	}
	return out
}

func (p *onepasswordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVaultDataSource,
		NewVaultsDataSource,
		NewItemDataSource,
		NewItemsDataSource,
		NewGroupDataSource,
		NewEnvironmentVariablesDataSource,
	}
}

func (p *onepasswordProvider) Actions(_ context.Context) []func() action.Action {
	return []func() action.Action{
		NewGeneratePasswordAction,
	}
}

func (p *onepasswordProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewSecretRefEphemeralResource,
	}
}

// defaultOPAccountName returns the local `op` CLI's default account UUID,
// or "" when the CLI is absent or lists no accounts.
var defaultOPAccountName = sync.OnceValue(func() string {
	out, err := exec.Command("op", "account", "list", "--format", "json").Output()
	if err != nil {
		return ""
	}
	return firstOPAccountUUID(out)
})

func firstOPAccountUUID(data []byte) string {
	var accounts []struct {
		AccountUUID string `json:"account_uuid"`
	}
	if json.Unmarshal(data, &accounts) != nil || len(accounts) == 0 {
		return ""
	}
	return accounts[0].AccountUUID
}
