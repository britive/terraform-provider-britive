package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceTypeResource is the resource implementation.
type ResourceTypeResource struct {
	client *britive.Client
}

// ResourceTypeResourceModel describes the resource data model.
type ResourceTypeResourceModel struct {
	ID          types.String                 `tfsdk:"id"`
	Name        types.String                 `tfsdk:"name"`
	Description types.String                 `tfsdk:"description"`
	Icon        types.String                 `tfsdk:"icon"`
	Parameters  []ResourceTypeParameterModel `tfsdk:"parameters"`
}

type ResourceTypeParameterModel struct {
	ParamName   types.String `tfsdk:"param_name"`
	ParamType   types.String `tfsdk:"param_type"`
	IsMandatory types.Bool   `tfsdk:"is_mandatory"`
}

// NewResourceTypeResource is a helper function to simplify the provider implementation.
func NewResourceTypeResource() resource.Resource {
	return &ResourceTypeResource{}
}

// Metadata returns the resource type name.
func (r *ResourceTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_manager_resource_type"
}

// Schema defines the schema for the resource.
func (r *ResourceTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Britive resource manager resource type",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource type ID",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of Britive resource type",
				Required:    true,
				Validators: []validator.String{
					validators.Alphanumeric(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the Britive resource type",
				Optional:    true,
			},
			"icon": schema.StringAttribute{
				Description: "Icon of Britive resource type (SVG format)",
				Optional:    true,
				Validators: []validator.String{
					validators.SVG(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"parameters": schema.SetNestedBlock{
				Description: "Parameters/Fields of the resource type",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"param_name": schema.StringAttribute{
							Description: "The parameter name",
							Required:    true,
							Validators: []validator.String{
								validators.Alphanumeric(),
							},
						},
						"param_type": schema.StringAttribute{
							Description: "The parameter type (string, password, ip-cidr, regex-pattern, list)",
							Required:    true,
						},
						"is_mandatory": schema.BoolAttribute{
							Description: "Whether the parameter is mandatory",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ResourceTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*britive.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *britive.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *ResourceTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map resource to API model
	resourceType := r.mapResourceToModel(&plan)

	log.Printf("[INFO] Adding new resource type: %#v", resourceType)

	// Create resource type
	rto, err := r.client.CreateResourceType(resourceType)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Submitted new resource type: %#v", rto)

	// Add icon if provided
	if !plan.Icon.IsNull() && plan.Icon.ValueString() != "" {
		log.Printf("[INFO] Adding icon to resource type: %#v", rto)
		userSVG := plan.Icon.ValueString()
		err = r.client.AddRemoveIcon(rto.ResourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Icon", err.Error())
			// Cleanup: delete the resource type if icon add fails
			if delErr := r.client.DeleteResourceType(rto.ResourceTypeID); delErr != nil {
				resp.Diagnostics.AddError("Error Cleaning Up Resource Type", delErr.Error())
			}
			return
		}
		log.Printf("[INFO] Added icon to resource type: %#v", rto)
	}

	// Set ID
	plan.ID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", rto.ResourceTypeID))

	// Read back to get computed values
	resourceTypeRead, err := r.client.GetResourceType(rto.ResourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(resourceTypeRead, &plan, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ResourceTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Reading resource type %s", resourceTypeID)

	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Received resource type %#v", resourceType)

	// Map model to resource
	r.mapModelToResource(resourceType, &state, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ResourceTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceTypeResourceModel
	var state ResourceTypeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	// Check for changes in main fields
	if !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!parametersEqual(plan.Parameters, state.Parameters) {

		resourceType := r.mapResourceToModel(&plan)

		log.Printf("[INFO] Updating resource type: %#v", resourceType)

		ur, err := r.client.UpdateResourceType(resourceType, resourceTypeID)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Resource Type", err.Error())
			return
		}

		log.Printf("[INFO] Updated resource type: %#v", ur)
	}

	// Handle icon separately
	if !plan.Icon.Equal(state.Icon) {
		log.Printf("[INFO] Updating icon to resource type: %#v", resourceTypeID)
		userSVG := plan.Icon.ValueString()
		err = r.client.AddRemoveIcon(resourceTypeID, userSVG)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Icon", err.Error())
			return
		}
		log.Printf("[INFO] Added icon to resource type: %#v", resourceTypeID)
	}

	// Read back to get updated values
	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Resource Type", err.Error())
		return
	}

	// Map model to resource
	r.mapModelToResource(resourceType, &plan, false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ResourceTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceTypeID, err := r.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Resource ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting resource type: %s", resourceTypeID)

	err = r.client.DeleteResourceType(resourceTypeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Resource Type", err.Error())
		return
	}

	log.Printf("[INFO] Resource type %s deleted", resourceTypeID)
}

// ImportState imports the resource state.
func (r *ResourceTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support format: resource-manager/resource-types/{id}
	importID := req.ID

	if !strings.HasPrefix(importID, "resource-manager/resource-types/") {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/resource-types/{id}', got: %s", importID),
		)
		return
	}

	parts := strings.Split(importID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'resource-manager/resource-types/{id}', got: %s", importID),
		)
		return
	}

	resourceTypeID := parts[2]
	if strings.TrimSpace(resourceTypeID) == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Resource type ID cannot be empty")
		return
	}

	log.Printf("[INFO] Importing resource type: %s", resourceTypeID)

	resourceType, err := r.client.GetResourceType(resourceTypeID)
	if errors.Is(err, britive.ErrNotFound) {
		resp.Diagnostics.AddError("Resource Type Not Found", fmt.Sprintf("Resource type %s not found", resourceTypeID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Resource Type", err.Error())
		return
	}

	// Set the state
	var state ResourceTypeResourceModel
	state.ID = types.StringValue(fmt.Sprintf("resource-manager/resource-types/%s", resourceType.ResourceTypeID))

	// Map model to resource (imported = true to preserve API param types)
	r.mapModelToResource(resourceType, &state, true)

	log.Printf("[INFO] Imported resource type: %s", resourceTypeID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper functions

func (r *ResourceTypeResource) mapResourceToModel(plan *ResourceTypeResourceModel) britive.ResourceType {
	resourceType := britive.ResourceType{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Parameters:  make([]britive.Parameter, 0),
	}

	for _, param := range plan.Parameters {
		resourceType.Parameters = append(resourceType.Parameters, britive.Parameter{
			ParamName:   param.ParamName.ValueString(),
			ParamType:   strings.ToLower(param.ParamType.ValueString()),
			IsMandatory: param.IsMandatory.ValueBool(),
		})
	}

	return resourceType
}

func (r *ResourceTypeResource) mapModelToResource(resourceType *britive.ResourceType, state *ResourceTypeResourceModel, imported bool) {
	state.Name = types.StringValue(resourceType.Name)
	state.Description = types.StringValue(resourceType.Description)

	// Build map of user's param types to preserve case (unless imported)
	paramMap := make(map[string]string)
	if !imported {
		for _, param := range state.Parameters {
			paramMap[param.ParamName.ValueString()] = param.ParamType.ValueString()
		}
	}

	// Map parameters
	parameters := make([]ResourceTypeParameterModel, 0)
	for _, param := range resourceType.Parameters {
		paramType := param.ParamType
		if !imported {
			// Use the user's original case for param_type if it exists in state
			if userType, ok := paramMap[param.ParamName]; ok {
				paramType = userType
			}
		}

		parameters = append(parameters, ResourceTypeParameterModel{
			ParamName:   types.StringValue(param.ParamName),
			ParamType:   types.StringValue(paramType),
			IsMandatory: types.BoolValue(param.IsMandatory),
		})
	}

	state.Parameters = parameters
}

func (r *ResourceTypeResource) parseUniqueID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 3 {
		return "", errs.NewInvalidResourceIDError("resource type", id)
	}
	return parts[2], nil
}

// parametersEqual compares two slices of ResourceTypeParameterModel for equality
func parametersEqual(a, b []ResourceTypeParameterModel) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison (order doesn't matter for sets)
	aMap := make(map[string]ResourceTypeParameterModel)
	for _, param := range a {
		key := param.ParamName.ValueString()
		aMap[key] = param
	}

	bMap := make(map[string]ResourceTypeParameterModel)
	for _, param := range b {
		key := param.ParamName.ValueString()
		bMap[key] = param
	}

	if len(aMap) != len(bMap) {
		return false
	}

	for key, aParam := range aMap {
		bParam, ok := bMap[key]
		if !ok {
			return false
		}
		// Compare all fields (case-insensitive for param_type)
		if !strings.EqualFold(aParam.ParamType.ValueString(), bParam.ParamType.ValueString()) ||
			aParam.IsMandatory.ValueBool() != bParam.IsMandatory.ValueBool() {
			return false
		}
	}

	return true
}
