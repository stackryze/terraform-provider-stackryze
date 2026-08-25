package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackryze/terraform-provider-stackryze/internal/client"
)

var _ resource.Resource = &recordResource{}
var _ resource.ResourceWithImportState = &recordResource{}

type recordResource struct {
	client *client.Client
}

type recordModel struct {
	ID      types.String `tfsdk:"id"`
	ZoneID  types.String `tfsdk:"zone_id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Content types.String `tfsdk:"content"`
	TTL     types.Int64  `tfsdk:"ttl"`
}

func NewRecordResource() resource.Resource { return &recordResource{} }

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A single DNS record within a Stackryze zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite id: `zone_id/type/name/content`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zone id (see the `stackryze_zone` data source).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Record label relative to the zone (`@` for apex).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Record type (A, AAAA, CNAME, MX, TXT, SRV, CAA).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Record value. For MX/SRV include priority, e.g. `10 mail.example.com`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ttl": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "TTL in seconds (min 3600).",
			},
		},
	}
}

func (r *recordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected *client.Client.")
		return
	}
	r.client = c
}

func recordID(zoneID, recType, name, content string) string {
	return fmt.Sprintf("%s/%s/%s/%s", zoneID, recType, name, content)
}

func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ttl := int(plan.TTL.ValueInt64())
	if ttl < 3600 {
		ttl = 3600
	}
	rec := client.Record{
		Name:    plan.Name.ValueString(),
		Type:    plan.Type.ValueString(),
		Content: plan.Content.ValueString(),
		TTL:     ttl,
	}
	if err := r.client.AddRecord(plan.ZoneID.ValueString(), rec); err != nil {
		resp.Diagnostics.AddError("Failed to create record", err.Error())
		return
	}

	plan.TTL = types.Int64Value(int64(ttl))
	plan.ID = types.StringValue(recordID(plan.ZoneID.ValueString(), rec.Type, rec.Name, rec.Content))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.FindRecord(state.ZoneID.ValueString(), state.Name.ValueString(), state.Type.ValueString(), state.Content.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read record", err.Error())
		return
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.TTL = types.Int64Value(int64(found.TTL))
	state.ID = types.StringValue(recordID(state.ZoneID.ValueString(), state.Type.ValueString(), state.Name.ValueString(), state.Content.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Only ttl is updatable in place; identity fields force replacement.
func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ttl := int(plan.TTL.ValueInt64())
	if ttl < 3600 {
		ttl = 3600
	}
	rec := client.Record{
		Name:    plan.Name.ValueString(),
		Type:    plan.Type.ValueString(),
		Content: plan.Content.ValueString(),
		TTL:     ttl,
	}
	// Re-issuing the record updates its TTL (PowerDNS replaces the rrset).
	if err := r.client.AddRecord(plan.ZoneID.ValueString(), rec); err != nil {
		resp.Diagnostics.AddError("Failed to update record", err.Error())
		return
	}
	plan.TTL = types.Int64Value(int64(ttl))
	plan.ID = types.StringValue(recordID(plan.ZoneID.ValueString(), rec.Type, rec.Name, rec.Content))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rec := client.Record{
		Name:    state.Name.ValueString(),
		Type:    state.Type.ValueString(),
		Content: state.Content.ValueString(),
	}
	if err := r.client.DeleteRecord(state.ZoneID.ValueString(), rec); err != nil {
		resp.Diagnostics.AddError("Failed to delete record", err.Error())
	}
}

// Import format: zone_id/type/name/content
func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import id", "Expected zone_id/type/name/content")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
