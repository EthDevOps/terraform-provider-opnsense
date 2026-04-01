package firewall

import (
	"context"
	"errors"
	"fmt"

	"github.com/browningluke/opnsense-go/pkg/api"
	"github.com/browningluke/opnsense-go/pkg/errs"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// dnatEndpoint represents source/destination nested objects in the DNAT API.
type dnatEndpoint struct {
	Network string `json:"network"`
	Port    string `json:"port"`
	Not     string `json:"not"`
}

// DNAT represents a Destination NAT rule in OPNsense.
type DNAT struct {
	Disabled      string          `json:"disabled"`
	NoRdr         string          `json:"nordr"`
	Sequence      string          `json:"sequence"`
	Interface     api.SelectedMap `json:"interface"`
	IPProtocol    api.SelectedMap `json:"ipprotocol"`
	Protocol      api.SelectedMap `json:"protocol"`
	Source        dnatEndpoint    `json:"source"`
	Destination   dnatEndpoint    `json:"destination"`
	Target        string          `json:"target"`
	TargetPort    string          `json:"local-port"`
	Log           string          `json:"log"`
	Description   string          `json:"descr"`
	PoolOptions   api.SelectedMap `json:"poolopts"`
	NatReflection api.SelectedMap `json:"natreflection"`
	Pass          api.SelectedMap     `json:"pass"`
	Category      api.SelectedMapList `json:"category"`
	Categories    string              `json:"categories"`
}

var dnatOpts = api.ReqOpts{
	AddEndpoint:         "/firewall/d_nat/addRule",
	GetEndpoint:         "/firewall/d_nat/getRule",
	UpdateEndpoint:      "/firewall/d_nat/setRule",
	DeleteEndpoint:      "/firewall/d_nat/delRule",
	ReconfigureEndpoint: "/firewall/d_nat/apply",
	Monad:               "rule",
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &dnatResource{}
var _ resource.ResourceWithConfigure = &dnatResource{}
var _ resource.ResourceWithImportState = &dnatResource{}

func newDNATResource() resource.Resource {
	return &dnatResource{}
}

// dnatResource defines the resource implementation.
type dnatResource struct {
	client *api.Client
}

func (r *dnatResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_dnat"
}

func (r *dnatResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = dnatResourceSchema()
}

func (r *dnatResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = apiClient
}

func (r *dnatResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *dnatResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert TF schema to OPNsense struct
	dnatRule, err := convertDNATSchemaToStruct(r.client, ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to parse firewall dnat, got error: %s", err))
		return
	}

	// Add firewall dnat rule
	id, err := api.Add(r.client, ctx, dnatOpts, dnatRule)
	if err != nil {
		if id != "" {
			// Tag new resource with ID from OPNsense
			data.Id = types.StringValue(id)

			// Save data into Terraform state
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}

		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to create firewall dnat, got error: %s", err))
		return
	}

	// Tag new resource with ID from OPNsense
	data.Id = types.StringValue(id)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnatResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *dnatResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get firewall dnat from OPNsense API
	resourceStruct, err := api.Get(r.client, ctx, dnatOpts, &DNAT{}, data.Id.ValueString())
	if err != nil {
		var notFoundError *errs.NotFoundError
		if errors.As(err, &notFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("firewall dnat not present in remote, removing from state"))
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read firewall dnat, got error: %s", err))
		return
	}

	// Convert OPNsense struct to TF schema
	resourceModel, err := convertDNATStructToSchema(resourceStruct)
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read firewall dnat, got error: %s", err))
		return
	}

	// ID cannot be added by convert... func, have to add here
	resourceModel.Id = data.Id

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &resourceModel)...)
}

func (r *dnatResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *dnatResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert TF schema to OPNsense struct
	dnatRule, err := convertDNATSchemaToStruct(r.client, ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to parse firewall dnat, got error: %s", err))
		return
	}

	// Update firewall dnat rule
	err = api.Update(r.client, ctx, dnatOpts, dnatRule, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to update firewall dnat, got error: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnatResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *dnatResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := api.Delete(r.client, ctx, dnatOpts, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to delete firewall dnat, got error: %s", err))
		return
	}
}

func (r *dnatResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
