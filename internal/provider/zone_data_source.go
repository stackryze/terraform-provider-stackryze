package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stackryze/terraform-provider-stackryze/internal/client"
)

var _ datasource.DataSource = &zoneDataSource{}

type zoneDataSource struct {
	client *client.Client
}

type zoneModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

func NewZoneDataSource() datasource.DataSource { return &zoneDataSource{} }

func (d *zoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *zoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Stackryze zone id by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zone name, e.g. `example.com`.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zone id used by `stackryze_record`.",
			},
		},
	}
}

func (d *zoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected *client.Client.")
		return
	}
	d.client = c
}

func (d *zoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg zoneModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	zone, err := d.client.GetZoneByName(cfg.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Zone lookup failed", err.Error())
		return
	}
	cfg.ID = types.StringValue(zone.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
