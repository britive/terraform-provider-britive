package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceLabelResource struct {
	client *britive.Client
}

type ResourceLabelResourceModel struct {
	ID          types.String              `tfsdk:"id"`
	Name        types.String              `tfsdk:"name"`
	Description types.String              `tfsdk:"description"`
	Internal    types.Bool                `tfsdk:"internal"`
	LabelColor  types.String              `tfsdk:"label_color"`
	Values      []ResourceLabelValueModel `tfsdk:"values"`
}

type ResourceLabelValueModel struct {
	ValueID     types.String `tfsdk:"value_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewResourceLabelResource() resource.Resource {
	return &ResourceLabelResource{}
}

func (r *ResourceLabelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_label"
}

func (r *ResourceLabelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource label",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validators.Alphanumeric(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"internal": schema.BoolAttribute{
				Computed: true,
			},
			"label_color": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"values": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value_id": schema.StringAttribute{
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"description": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *ResourceLabelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *britive.Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *ResourceLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	label := britive.ResourceLabel{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		LabelColor:  plan.LabelColor.ValueString(),
		Values:      make([]britive.ResourceLabelValue, 0),
	}

	for _, v := range plan.Values {
		label.Values = append(label.Values, britive.ResourceLabelValue{
			Name:        v.Name.ValueString(),
			Description: v.Description.ValueString(),
		})
	}

	log.Printf("[INFO] Creating resource label: %#v", label)

	created, err := r.client.CreateUpdateResourceLabel(label, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Label", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/labels/%s", created.LabelId))
	plan.Internal = types.BoolValue(created.Internal)

	var values []ResourceLabelValueModel
	for _, v := range created.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	plan.Values = values

	log.Printf("[INFO] Created resource label: %s", created.LabelId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())
	label, err := r.client.GetResourceLabel(labelID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Label", err.Error())
		return
	}

	state.Name = types.StringValue(label.Name)
	state.Description = types.StringValue(label.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = types.StringValue(label.LabelColor)

	var values []ResourceLabelValueModel
	for _, v := range label.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	state.Values = values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceLabelResourceModel
	var state ResourceLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())

	label := britive.ResourceLabel{
		LabelId:     labelID,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		LabelColor:  plan.LabelColor.ValueString(),
		Values:      make([]britive.ResourceLabelValue, 0),
	}

	for _, v := range plan.Values {
		labelValue := britive.ResourceLabelValue{
			Name:        v.Name.ValueString(),
			Description: v.Description.ValueString(),
		}
		if !v.ValueID.IsNull() {
			labelValue.ValueId = v.ValueID.ValueString()
		}
		label.Values = append(label.Values, labelValue)
	}

	log.Printf("[INFO] Updating resource label: %s", labelID)

	updated, err := r.client.CreateUpdateResourceLabel(label, true)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Resource Label", err.Error())
		return
	}

	plan.Internal = types.BoolValue(updated.Internal)

	var values []ResourceLabelValueModel
	for _, v := range updated.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	plan.Values = values

	log.Printf("[INFO] Updated resource label: %s", labelID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := parseLabelID(state.ID.ValueString())

	log.Printf("[INFO] Deleting resource label: %s", labelID)

	err := r.client.DeleteResourceLabel(labelID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Label", err.Error())
		return
	}

	log.Printf("[INFO] Deleted resource label: %s", labelID)
}

func (r *ResourceLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := req.ID
	var labelID string

	if strings.HasPrefix(importID, "resource-manager/labels/") {
		parts := strings.Split(importID, "/")
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Import ID must be 'resource-manager/labels/{id}' or '{id}', got: %s", importID))
			return
		}
		labelID = parts[2]
	} else {
		labelID = importID
	}

	label, err := r.client.GetResourceLabel(labelID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Label Not Found", fmt.Sprintf("Label %s not found", labelID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Label", err.Error())
		return
	}

	var state ResourceLabelResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/labels/%s", label.LabelId))
	state.Name = types.StringValue(label.Name)
	state.Description = types.StringValue(label.Description)
	state.Internal = types.BoolValue(label.Internal)
	state.LabelColor = types.StringValue(label.LabelColor)

	var values []ResourceLabelValueModel
	for _, v := range label.Values {
		values = append(values, ResourceLabelValueModel{
			ValueID:     types.StringValue(v.ValueId),
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	state.Values = values

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func parseLabelID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return id
}
