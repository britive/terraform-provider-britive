package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/britive/terraform-provider-britive/britive_client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ResourceResourceManagerResourceBrokerPool{}
	_ resource.ResourceWithConfigure   = &ResourceResourceManagerResourceBrokerPool{}
	_ resource.ResourceWithImportState = &ResourceResourceManagerResourceBrokerPool{}
)

type ResourceResourceManagerResourceBrokerPool struct {
	client       *britive_client.Client
	helper       *ResourceResourceManagerResourceBrokerPoolHelper
	importHelper *imports.ImportHelper
}

type ResourceResourceManagerResourceBrokerPoolHelper struct{}

func NewResourceResourceManagerResourceBrokerPool() resource.Resource {
	return &ResourceResourceManagerResourceBrokerPool{}
}

func NewResourceResourceManagerResourceBrokerPoolHelper() *ResourceResourceManagerResourceBrokerPoolHelper {
	return &ResourceResourceManagerResourceBrokerPoolHelper{}
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "britive_resource_manager_resource_broker_pools"
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	tflog.Info(ctx, "Configuring the Britive Resource Manager Resource Broker Pool resource")

	if req.ProviderData == nil {
		return
	}

	rbp.client = req.ProviderData.(*britive_client.Client)
	if rbp.client == nil {
		resp.Diagnostics.AddError(
			"Provider client error",
			"REST API client is not configured",
		)
		tflog.Error(ctx, "Provider client is nil after Configure", map[string]interface{}{
			"method": "Configure",
		})
		return
	}

	tflog.Info(ctx, "Provider client configured")
	rbp.helper = NewResourceResourceManagerResourceBrokerPoolHelper()
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema for Britive Resource Broker Pool resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of resource",
				Validators: []validator.String{
					validate.StringFunc(
						"resource_id",
						validate.StringIsNotWhiteSpace(),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"broker_pools": schema.SetAttribute{
				Required:    true,
				Description: "The broker pool names to be associated to the resource",
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Called create for britive_resource_manager_resource_broker_pools")

	var plan britive_client.ResourceManagerResourceBrokerPoolsPlan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan during resource broker pools creation", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	serverAccessResourceID := plan.ResourceID.ValueString()
	serverAccessResourceName, err := rbp.client.GetResourceName(ctx, serverAccessResourceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource broker pools", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to get resource name, error:%#v", err))
		return
	}

	brokerPoolNamesString, err := rbp.helper.mapBrokerPoolsToList(ctx, plan.BrokerPools)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource broker pools", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to create resource broker pools, error:%#v", err))
		return
	}

	err = rbp.client.AddBrokerPoolsResource(ctx, brokerPoolNamesString, serverAccessResourceName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource broker pools", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to add broker pools, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Submitted new broker pools %#v for the resource: %#v", brokerPoolNamesString, serverAccessResourceID))
	plan.ID = types.StringValue(rbp.helper.generateUniqueID(serverAccessResourceID))

	planPtr, err := rbp.helper.getAndMapModelToPlan(ctx, plan, *rbp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource broker pools model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after create", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Create completed and state set", map[string]interface{}{
		"resource": planPtr,
	})
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Read called for britive_resource_manager_resource_broker_pools")

	if rbp.client == nil {
		resp.Diagnostics.AddError("Client not found", "")
		tflog.Error(ctx, "Read failed: client is nil")
		return
	}

	var state britive_client.ResourceManagerResourceBrokerPoolsPlan
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Read failed to get resource state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	newPlan, err := rbp.helper.getAndMapModelToPlan(ctx, state, *rbp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get resource broker pools",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "get and map resource model to plan failed in Read", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	diags = resp.State.Set(ctx, newPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state in Read", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, "Read completed for britive_resource_manager_resource_broker_pools")
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (rbp *ResourceResourceManagerResourceBrokerPool) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Delete called for britive_resource_manager_resource_broker_pools")

	var state britive_client.ResourceManagerResourceBrokerPoolsPlan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Delete failed to get state", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	serverAccessResourceID, err := rbp.helper.parseUniqueID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete resource broker pools", err.Error())
		tflog.Error(ctx, "Failed to parse resourceBrokerPoolsID")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Deleting the broker pools for resource: %s", serverAccessResourceID))

	err = rbp.client.DeleteBrokerPoolsResource(ctx, serverAccessResourceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete broker pools", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to delete broker pools, error:%#v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Broker pools for the resource %s are deleted", serverAccessResourceID))

	resp.State.RemoveResource(ctx)
}

func (rbp *ResourceResourceManagerResourceBrokerPool) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importId := req.ID

	importData := &imports.ImportHelperData{
		ID: importId,
	}

	if err := rbp.importHelper.ParseImportID([]string{"resources/(?P<name>[^/]+)/broker-pools", "(?P<name>[^/]+)/broker-pools"}, importData); err != nil {
		resp.Diagnostics.AddError("Failed to import resource broker pools", err.Error())
		tflog.Error(ctx, fmt.Sprintf("Failed to parse ImportID, error:%#v", err))
		return
	}
	serverAccessResourceName := importData.Fields["name"]
	if strings.TrimSpace(serverAccessResourceName) == "" {
		resp.Diagnostics.AddError("Failed to import resource broker pools", "Invalid name")
		tflog.Error(ctx, "Failed to import resource broker pools, Invalid name")
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Importing broker pools for the resource : %s", serverAccessResourceName))

	plan := britive_client.ResourceManagerResourceBrokerPoolsPlan{
		ID: types.StringValue(rbp.helper.generateUniqueID(serverAccessResourceName)),
	}

	planPtr, err := rbp.helper.getAndMapModelToPlan(ctx, plan, *rbp.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set state after import",
			fmt.Sprintf("Error: %v", err),
		)
		tflog.Error(ctx, "Failed get and map resource broker pools model to plan", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, planPtr)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state after import", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Imported resource: %#v", planPtr))
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) getAndMapModelToPlan(ctx context.Context, plan britive_client.ResourceManagerResourceBrokerPoolsPlan, c britive_client.Client) (*britive_client.ResourceManagerResourceBrokerPoolsPlan, error) {
	serverAccessResourceID, err := rbph.parseUniqueID(plan.ID.ValueString())
	if err != nil {
		return nil, err
	}

	tflog.Info(ctx, fmt.Sprintf("Reading broker pools for resource %s", serverAccessResourceID))

	plan.ResourceID = types.StringValue(serverAccessResourceID)

	serverAccessResourceName, err := c.GetResourceName(ctx, serverAccessResourceID)
	if err != nil {
		return nil, err
	}

	brokerPoolNames, err := rbph.getBrokerPoolNames(ctx, serverAccessResourceName, c)
	if err != nil {
		return nil, err
	}

	brokerPoolsSet, err := rbph.mapListBrokerPoolsToSet(ctx, brokerPoolNames)
	if err != nil {
		return nil, err
	}

	plan.BrokerPools = brokerPoolsSet

	return &plan, nil
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) getBrokerPoolNames(ctx context.Context, serverAccessResourceName string, c britive_client.Client) (brokerPoolNames []string, err error) {
	brokerPools, err := c.GetBrokerPoolsResource(ctx, serverAccessResourceName)
	if errors.Is(err, britive_client.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("broker pools for resource %s", serverAccessResourceName)
	}
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Received broker pools %#v for resource %#v", brokerPools, serverAccessResourceName)

	for _, brokerPool := range *brokerPools {
		brokerPoolNames = append(brokerPoolNames, brokerPool.Name)
	}
	return brokerPoolNames, nil
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) generateUniqueID(serverAccessResourceID string) string {
	return fmt.Sprintf("resources/%s/broker-pools", serverAccessResourceID)
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) parseUniqueID(ID string) (serverAccessResourceID string, err error) {
	brokerPoolsResourceParts := strings.Split(ID, "/")
	if len(brokerPoolsResourceParts) < 3 {
		err = errs.NewInvalidResourceIDError("brokerPools", ID)
		return
	}

	serverAccessResourceID = brokerPoolsResourceParts[1]
	return
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) mapBrokerPoolsToList(ctx context.Context, set types.Set) ([]string, error) {
	var list []string
	diags := set.ElementsAs(ctx, &list, true)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to map broker pools as list")
	}

	return list, nil
}

func (rbph *ResourceResourceManagerResourceBrokerPoolHelper) mapListBrokerPoolsToSet(ctx context.Context, list []string) (types.Set, error) {
	set, diags := types.SetValueFrom(ctx, types.StringType, list)
	if diags.HasError() {
		return types.Set{}, fmt.Errorf("failed to map broker pools as set")
	}

	return set, nil
}
