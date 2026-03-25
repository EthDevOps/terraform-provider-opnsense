package firewall

import (
	"regexp"

	"github.com/browningluke/opnsense-go/pkg/api"
	"github.com/browningluke/terraform-provider-opnsense/internal/tools"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dnatResourceModel describes the resource data model.
type dnatResourceModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	NoRdr   types.Bool `tfsdk:"no_rdr"`

	Sequence  types.Int64  `tfsdk:"sequence"`
	Interface types.String `tfsdk:"interface"`

	IPProtocol types.String `tfsdk:"ip_protocol"`
	Protocol   types.String `tfsdk:"protocol"`

	Source      *firewallLocation `tfsdk:"source"`
	Destination *firewallLocation `tfsdk:"destination"`
	Target      *firewallTarget   `tfsdk:"target"`

	Log           types.Bool   `tfsdk:"log"`
	Description   types.String `tfsdk:"description"`
	PoolOptions   types.String `tfsdk:"pool_options"`
	NatReflection types.String `tfsdk:"nat_reflection"`
	Pass          types.String `tfsdk:"pass"`

	Id types.String `tfsdk:"id"`
}

func dnatResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Destination NAT (port forwarding) allows redirecting incoming traffic to an internal host. This is used to make services on internal machines accessible from the outside.",

		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable this firewall DNAT rule. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"no_rdr": schema.BoolAttribute{
				MarkdownDescription: "Enabling this option will disable redirection for traffic matching this rule. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sequence": schema.Int64Attribute{
				MarkdownDescription: "Specify the order of this DNAT rule. Defaults to `1`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"interface": schema.StringAttribute{
				MarkdownDescription: "Choose on which interface(s) packets must come in to match this rule.",
				Required:            true,
			},
			"ip_protocol": schema.StringAttribute{
				MarkdownDescription: "Select the Internet Protocol version this rule applies to. Available values: `inet`, `inet6`. Defaults to `inet`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("inet", "inet6"),
				},
				Default: stringdefault.StaticString("inet"),
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Choose which IP protocol this rule should match.",
				Required:            true,
			},
			"source": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"net":    types.StringType,
							"port":   types.StringType,
							"invert": types.BoolType,
						},
						map[string]attr.Value{
							"net":    types.StringValue("any"),
							"port":   types.StringValue(""),
							"invert": types.BoolValue(false),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"net": schema.StringAttribute{
						MarkdownDescription: "Specify the IP address, CIDR or alias for the source of the packet for this mapping. For `<INT> net`, enter `<int>` (e.g. `lan`). For `<INT> address`, enter `<int>ip` (e.g. `lanip`). Defaults to `any`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("any"),
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "Specify the source port for this rule. This is usually random and almost never equal to the destination port range (and should usually be `\"\"`). Defaults to `\"\"`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile("^(\\d|-)+$|^([a-z])+$"),
								"must be number (80), range (80-443) or well known name (http)"),
						},
					},
					"invert": schema.BoolAttribute{
						MarkdownDescription: "Use this option to invert the sense of the match. Defaults to `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"destination": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"net":    types.StringType,
							"port":   types.StringType,
							"invert": types.BoolType,
						},
						map[string]attr.Value{
							"net":    types.StringValue("any"),
							"port":   types.StringValue(""),
							"invert": types.BoolValue(false),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"net": schema.StringAttribute{
						MarkdownDescription: "Specify the IP address, CIDR or alias for the destination of the packet for this mapping. For `<INT> net`, enter `<int>` (e.g. `lan`). For `<INT> address`, enter `<int>ip` (e.g. `lanip`). Defaults to `any`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("any"),
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "Destination port number or well known name (imap, imaps, http, https, ...), for ranges use a dash. Defaults to `\"\"`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile("^(\\d|-)+$|^(\\w){0,32}$"),
								"must be number (80), range (80-443), well known name (http) or alias name"),
						},
					},
					"invert": schema.BoolAttribute{
						MarkdownDescription: "Use this option to invert the sense of the match. Defaults to `false`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"target": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "Specify the IP address or alias for the packets to be redirected to. For `<INT> address`, enter `<int>ip` (e.g. `lanip`).",
						Required:            true,
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "Destination port number or well known name (imap, imaps, http, https, ...), for ranges use a dash. Defaults to `\"\"`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile("^(\\d|-)+$|^([a-z])+$"),
								"must be number (80), range (80-443) or well known name (http)"),
						},
					},
				},
			},
			"log": schema.BoolAttribute{
				MarkdownDescription: "Log packets that are handled by this rule. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description here for your reference (not parsed). Must be between 1 and 255 characters.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"pool_options": schema.StringAttribute{
				MarkdownDescription: "Load balancing pool options for multiple target addresses. Available values: `\"\"` (default), `round-robin`, `round-robin sticky-address`, `random`, `random sticky-address`, `source-hash`, `bitmask`. Defaults to `\"\"`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"nat_reflection": schema.StringAttribute{
				MarkdownDescription: "NAT reflection mode. Available values: `\"\"` (system default), `enable`, `disable`. Defaults to `\"\"`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"pass": schema.StringAttribute{
				MarkdownDescription: "When set, a firewall rule matching this NAT rule will be automatically created. Available values: `\"\"` (none), `pass`. Defaults to `\"\"`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func dnatDataSourceSchema() dschema.Schema {
	return dschema.Schema{
		MarkdownDescription: "Destination NAT (port forwarding) allows redirecting incoming traffic to an internal host. This is used to make services on internal machines accessible from the outside.",

		Attributes: map[string]dschema.Attribute{
			"id": dschema.StringAttribute{
				MarkdownDescription: "UUID of the resource.",
				Required:            true,
			},
			"enabled": dschema.BoolAttribute{
				MarkdownDescription: "Enable this firewall DNAT rule.",
				Computed:            true,
			},
			"no_rdr": dschema.BoolAttribute{
				MarkdownDescription: "Enabling this option will disable redirection for traffic matching this rule.",
				Computed:            true,
			},
			"sequence": dschema.Int64Attribute{
				MarkdownDescription: "Specify the order of this DNAT rule.",
				Computed:            true,
			},
			"interface": dschema.StringAttribute{
				MarkdownDescription: "The interface on which packets must come in to match this rule.",
				Computed:            true,
			},
			"ip_protocol": dschema.StringAttribute{
				MarkdownDescription: "Select the Internet Protocol version this rule applies to. Available values: `inet`, `inet6`.",
				Computed:            true,
			},
			"protocol": dschema.StringAttribute{
				MarkdownDescription: "Choose which IP protocol this rule should match.",
				Computed:            true,
			},
			"source": dschema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]dschema.Attribute{
					"net": dschema.StringAttribute{
						MarkdownDescription: "Specify the IP address, CIDR or alias for the source of the packet for this mapping.",
						Computed:            true,
					},
					"port": dschema.StringAttribute{
						MarkdownDescription: "Specify the source port for this rule.",
						Computed:            true,
					},
					"invert": dschema.BoolAttribute{
						MarkdownDescription: "Use this option to invert the sense of the match.",
						Computed:            true,
					},
				},
			},
			"destination": dschema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]dschema.Attribute{
					"net": dschema.StringAttribute{
						MarkdownDescription: "Specify the IP address, CIDR or alias for the destination of the packet for this mapping.",
						Computed:            true,
					},
					"port": dschema.StringAttribute{
						MarkdownDescription: "Specify the port for the destination of the packet for this mapping.",
						Computed:            true,
					},
					"invert": dschema.BoolAttribute{
						MarkdownDescription: "Use this option to invert the sense of the match.",
						Computed:            true,
					},
				},
			},
			"target": dschema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]dschema.Attribute{
					"ip": dschema.StringAttribute{
						MarkdownDescription: "Specify the IP address or alias for the packets to be redirected to.",
						Computed:            true,
					},
					"port": dschema.StringAttribute{
						MarkdownDescription: "Destination port number or well known name (imap, imaps, http, https, ...), for ranges use a dash.",
						Computed:            true,
					},
				},
			},
			"log": dschema.BoolAttribute{
				MarkdownDescription: "Log packets that are handled by this rule.",
				Computed:            true,
			},
			"description": dschema.StringAttribute{
				MarkdownDescription: "Optional description here for your reference (not parsed).",
				Computed:            true,
			},
			"pool_options": dschema.StringAttribute{
				MarkdownDescription: "Load balancing pool options for multiple target addresses.",
				Computed:            true,
			},
			"nat_reflection": dschema.StringAttribute{
				MarkdownDescription: "NAT reflection mode.",
				Computed:            true,
			},
			"pass": dschema.StringAttribute{
				MarkdownDescription: "When set, a firewall rule matching this NAT rule will be automatically created.",
				Computed:            true,
			},
		},
	}
}

func convertDNATSchemaToStruct(d *dnatResourceModel) (*DNAT, error) {
	return &DNAT{
		Disabled:   tools.BoolToString(!d.Enabled.ValueBool()),
		NoRdr:      tools.BoolToString(d.NoRdr.ValueBool()),
		Sequence:   tools.Int64ToString(d.Sequence.ValueInt64()),
		Interface:  api.SelectedMap(d.Interface.ValueString()),
		IPProtocol: api.SelectedMap(d.IPProtocol.ValueString()),
		Protocol:   api.SelectedMap(d.Protocol.ValueString()),
		Source: dnatEndpoint{
			Network: d.Source.Net.ValueString(),
			Port:    d.Source.Port.ValueString(),
			Not:     tools.BoolToString(d.Source.Invert.ValueBool()),
		},
		Destination: dnatEndpoint{
			Network: d.Destination.Net.ValueString(),
			Port:    d.Destination.Port.ValueString(),
			Not:     tools.BoolToString(d.Destination.Invert.ValueBool()),
		},
		Target:        d.Target.IP.ValueString(),
		TargetPort:    d.Target.Port.ValueString(),
		Log:           tools.BoolToString(d.Log.ValueBool()),
		Description:   d.Description.ValueString(),
		PoolOptions:   api.SelectedMap(d.PoolOptions.ValueString()),
		NatReflection: api.SelectedMap(d.NatReflection.ValueString()),
		Pass:          api.SelectedMap(d.Pass.ValueString()),
	}, nil
}

func convertDNATStructToSchema(d *DNAT) (*dnatResourceModel, error) {
	return &dnatResourceModel{
		Enabled:    types.BoolValue(!tools.StringToBool(d.Disabled)),
		NoRdr:      types.BoolValue(tools.StringToBool(d.NoRdr)),
		Sequence:   tools.StringToInt64Null(d.Sequence),
		Interface:  types.StringValue(d.Interface.String()),
		IPProtocol: types.StringValue(d.IPProtocol.String()),
		Protocol:   types.StringValue(d.Protocol.String()),
		Source: &firewallLocation{
			Net:    types.StringValue(d.Source.Network),
			Port:   types.StringValue(d.Source.Port),
			Invert: types.BoolValue(tools.StringToBool(d.Source.Not)),
		},
		Destination: &firewallLocation{
			Net:    types.StringValue(d.Destination.Network),
			Port:   types.StringValue(d.Destination.Port),
			Invert: types.BoolValue(tools.StringToBool(d.Destination.Not)),
		},
		Target: &firewallTarget{
			IP:   types.StringValue(d.Target),
			Port: types.StringValue(d.TargetPort),
		},
		Log:           types.BoolValue(tools.StringToBool(d.Log)),
		Description:   tools.StringOrNull(d.Description),
		PoolOptions:   types.StringValue(d.PoolOptions.String()),
		NatReflection: types.StringValue(d.NatReflection.String()),
		Pass:          types.StringValue(d.Pass.String()),
	}, nil
}
