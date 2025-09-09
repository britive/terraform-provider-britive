package resourcemanager

import (
	"context"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceServerAccess - Terraform Resource for Server Access Resource
type ResourceServerAccess struct {
	Resource     *schema.Resource
	helper       *ResourceServerAccessHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceServerAccess - Initializes new server access resource
func NewResourceServerAccess(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceServerAccess {
	rsa := &ResourceServerAccess{
		helper:       NewResourceServerAccessHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rsa.Resource = &schema.Resource{
		CreateContext: rsa.resourceCreate,
		ReadContext:   rsa.resourceRead,
		UpdateContext: rsa.resourceUpdate,
		DeleteContext: rsa.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rsa.resourceStateImporter,
		},
		CustomizeDiff: rsa.validation.ValidateImmutableFields([]string{
			"resource_type",
		}),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of server access resource",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The description of the server access resource",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"resource_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The resource type name associated with the server access resource",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"resource_type_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parameter_values": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The parameter values for the fields of the resource type",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"resource_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The resource labels associated with the server access resource",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
	return rsa
}

//region Resource Server Access Context Operations

func (rsa *ResourceServerAccess) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	serverAccessResource := britive.ServerAccessResource{}

	err := rsa.helper.mapResourceToModel(d, m, &serverAccessResource, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new server access resource: %#v", serverAccessResource)

	sa, err := c.AddServerAccessResource(serverAccessResource)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new server access resource: %#v", sa)
	d.SetId(sa.ResourceID)

	rsa.resourceRead(ctx, d, m)

	return diags
}

func (rsa *ResourceServerAccess) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := rsa.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rsa *ResourceServerAccess) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	serverAccessResourceID := d.Id()

	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("resource_type") || d.HasChange("parameter_values") || d.HasChange("resource_labels") {
		hasChanges = true
		serverAccessResource := britive.ServerAccessResource{}

		err := rsa.helper.mapResourceToModel(d, m, &serverAccessResource, true)
		if err != nil {
			return diag.FromErr(err)
		}

		ursa, err := c.UpdateServerAccessResource(serverAccessResource, serverAccessResourceID)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated server access resource: %#v", ursa)
		d.SetId(serverAccessResourceID)
	}
	if hasChanges {
		return rsa.resourceRead(ctx, d, m)
	}
	return nil
}

func (rsa *ResourceServerAccess) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	serverAccessResourceID := d.Id()
	log.Printf("[INFO] Deleting server access resource: %s", serverAccessResourceID)
	err := c.DeleteServerAccessResource(serverAccessResourceID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Resource %s deleted", serverAccessResourceID)
	d.SetId("")

	return diags
}

func (rsa *ResourceServerAccess) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rsa.importHelper.ParseImportID([]string{"resources/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	serverAccessResourceName := d.Get("name").(string)
	if strings.TrimSpace(serverAccessResourceName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing server access resource : %s", serverAccessResourceName)

	serverAccessResource, err := c.GetServerAccessResourceByName(serverAccessResourceName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("role %s", serverAccessResourceName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(serverAccessResource.ResourceID)

	err = rsa.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceServerAccessHelper - Resource Server Access helper functions
type ResourceServerAccessHelper struct {
}

// NewResourceServerAccessHelper - Initializes new server access resource helper
func NewResourceServerAccessHelper() *ResourceServerAccessHelper {
	return &ResourceServerAccessHelper{}
}

//region Resource Server Access helper functions

func (rrh *ResourceServerAccessHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, serverAccessResource *britive.ServerAccessResource, isUpdate bool) error {
	serverAccessResource.Name = d.Get("name").(string)
	serverAccessResource.Description = d.Get("description").(string)
	serverAccessResource.ResourceType.Name = d.Get("resource_type").(string)
	convertedStringMap := InterfaceMapToStringMap(d.Get("parameter_values").(map[string]interface{}))
	serverAccessResource.ResourceTypeParameterValues = convertedStringMap
	revertedSliceMap := ConvertToSliceMap(d.Get("resource_labels").(map[string]interface{}))
	revertedSliceMap["Resource-Type"] = []string{serverAccessResource.ResourceType.Name}
	serverAccessResource.ResourceLabels = revertedSliceMap
	if isUpdate {
		serverAccessResource.ResourceType.ResourceTypeID = d.Get("resource_type_id").(string)
	}

	return nil
}

func (rsah *ResourceServerAccessHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	serverAccessResourceID := d.Id()

	log.Printf("[INFO] Reading server access resource %s", serverAccessResourceID)

	serverAccessResource, err := c.GetServerAccessResource(serverAccessResourceID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("role %s", serverAccessResourceID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received server access resource %#v", serverAccessResource.Name)

	if err := d.Set("name", serverAccessResource.Name); err != nil {
		return err
	}
	if err := d.Set("description", serverAccessResource.Description); err != nil {
		return err
	}
	if err := d.Set("resource_type", serverAccessResource.ResourceType.Name); err != nil {
		return err
	}
	if err := d.Set("resource_type_id", serverAccessResource.ResourceType.ResourceTypeID); err != nil {
		return err
	}
	interfaceMap := StringMapToInterfaceMap(serverAccessResource.ResourceTypeParameterValues)
	if err := d.Set("parameter_values", interfaceMap); err != nil {
		return err
	}

	delete(serverAccessResource.ResourceLabels, "Resource-Type")
	stringMap := ConvertToStringMap(serverAccessResource.ResourceLabels)
	delete(stringMap, "Resource-Type")

	newResLabelsMap := d.Get("resource_labels").(map[string]interface{})
	delete(newResLabelsMap, "Resource-Type")
	if britive.ResourceLabelsMapEqual(stringMap, newResLabelsMap) {
		if err := d.Set("resource_labels", newResLabelsMap); err != nil {
			return err
		}
	} else if err := d.Set("resource_labels", stringMap); err != nil {
		return err
	}

	return nil
}

// Convert map[string][]string to map[string]string
func ConvertToStringMap(input map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range input {
		result[key] = strings.Join(value, ",")
	}
	return result
}

// Convert map[string]string to map[string][]string
func ConvertToSliceMap(input map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	for key, value := range input {
		result[key] = strings.Split(value.(string), ",")
	}
	return result
}

// Convert map[string]string to map[string]interface{}
func StringMapToInterfaceMap(input map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range input {
		result[key] = value
	}
	return result
}

// Convert map[string]interface{} to map[string]string
func InterfaceMapToStringMap(input map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range input {
		if str, ok := value.(string); ok {
			result[key] = str
		}
	}
	return result
}

func suppressCommaSeparatedDiffs(k, old, new string, d *schema.ResourceData) bool {
	normalize := func(s string) []string {
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		sort.Strings(parts)
		return parts
	}

	oldParts := normalize(old)
	newParts := normalize(new)

	return reflect.DeepEqual(oldParts, newParts)
}

//endregion
