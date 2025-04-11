package britive

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceServerAccess - Terraform Resource for Server Access Resource
type ResourceServerAccess struct {
	Resource     *schema.Resource
	helper       *ResourceServerAccessHelper
	validation   *Validation
	importHelper *ImportHelper
}

// NewResourceServerAccess - Initializes new server access resource
func NewResourceServerAccess(v *Validation, importHelper *ImportHelper) *ResourceServerAccess {
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
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
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
				ForceNew:     true,
				Description:  "The resource type associated with the server access resource",
				ValidateFunc: validation.StringIsNotWhiteSpace,
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
				// Elem: &schema.Schema{
				// 	Type: schema.TypeList,
				// 	Elem: &schema.Schema{
				// 		Type: schema.TypeString,
				// 	},
				// },
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
	d.SetId(rsa.helper.generateUniqueID(sa.ResourceID))

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

	serverAccessResourceID, err := rsa.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("resource_type") || d.HasChange("parameter_values") || d.HasChange("resource_labels") {
		hasChanges = true
		serverAccessResource := britive.ServerAccessResource{}

		err := rsa.helper.mapResourceToModel(d, m, &serverAccessResource, true)
		if err != nil {
			return diag.FromErr(err)
		}

		ursa, err := c.UpdateServerAccessResource(serverAccessResource)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated server access resource: %#v", ursa)
		d.SetId(rsa.helper.generateUniqueID(serverAccessResourceID))
	}
	if hasChanges {
		return rsa.resourceRead(ctx, d, m)
	}
	return nil
}

func (rsa *ResourceServerAccess) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	serverAccessResourceID, err := rsa.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting server access resource: %s", serverAccessResourceID)
	err = c.DeleteServerAccessResource(serverAccessResourceID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Role %s deleted", serverAccessResourceID)
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
		return nil, NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing server access resource : %s", serverAccessResourceName)

	serverAccessResource, err := c.GetServerAccessResourceByName(serverAccessResourceName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, NewNotFoundErrorf("role %s", serverAccessResourceName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rsa.helper.generateUniqueID(serverAccessResource.ResourceID))

	err = rsa.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	// log.Printf("[INFO] Imported server access resource: %s", serverAccessResourceName)
	// log.Printf("[INFO] Imported server access resource: %s", serverAccessResource)
	// log.Printf("[INFO] Imported server access resource: %#v", []*schema.ResourceData{d})

	// newServerAccessresource := expandMapOfLists(d.Get("resource_labels").(map[string]interface{}))

	// log.Printf("[INFO] AftercallingFunction: %#v", newServerAccessresource)

	// d.Set("resource_labels", newServerAccessresource)

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
	// resourceLabels := make(map[string][]string)
	// if v, ok := d.Get("resource_labels").(map[string]interface{}); ok {
	// 	for key, value := range v {
	// 		if strVal, ok := value.(string); ok {
	// 			resourceLabels[key] = strings.Split(strVal, ",")
	// 		}
	// 	}
	// }
	// Convert back to map[string][]string
	revertedSliceMap := convertToSliceMap(d.Get("resource_labels").(map[string]interface{}))
	serverAccessResource.ResourceLabels = revertedSliceMap

	return nil
}

func (rsah *ResourceServerAccessHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	serverAccessResourceID, err := rsah.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading server access resource %s", serverAccessResourceID)

	serverAccessResource, err := c.GetServerAccessResource(serverAccessResourceID)
	if errors.Is(err, britive.ErrNotFound) {
		return NewNotFoundErrorf("role %s", serverAccessResourceID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received server access resource %#v", serverAccessResource.Name)
	// log.Printf("[INFO] Received server access resource %#v", serverAccessResource)

	if err := d.Set("name", serverAccessResource.Name); err != nil {
		return err
	}
	if err := d.Set("description", serverAccessResource.Description); err != nil {
		return err
	}
	if err := d.Set("resource_type", serverAccessResource.ResourceType.Name); err != nil {
		return err
	}
	interfaceMap := StringMapToInterfaceMap(serverAccessResource.ResourceTypeParameterValues)
	if err := d.Set("parameter_values", interfaceMap); err != nil {
		return err
	}

	stringMap := convertToStringMap(serverAccessResource.ResourceLabels)
	if err := d.Set("resource_labels", stringMap); err != nil {
		return err
	}

	// // resourceLabelsMap := make(map[string]interface{})
	// // for key, values := range serverAccessResource.ResourceLabels {
	// // 	resourceLabelsMap[key] = strings.Join(values, ",")
	// // }

	// newServerAccessresource := expandMapOfLists(serverAccessResource.ResourceLabels)

	// log.Printf("[INFO] Sending to expand Map of Lists server access resource %s", serverAccessResource.ResourceLabels)

	return nil
}

func (resourceServerAccessHelper *ResourceServerAccessHelper) generateUniqueID(serverAccessResourceID string) string {
	return fmt.Sprintf("resources/%s", serverAccessResourceID)
}

func (resourceServerAccessHelper *ResourceServerAccessHelper) parseUniqueID(ID string) (serverAccessResourceID string, err error) {
	serverAccessResourceParts := strings.Split(ID, "/")
	if len(serverAccessResourceParts) < 2 {
		err = NewInvalidResourceIDError("serverAccessResource", ID)
		return
	}

	serverAccessResourceID = serverAccessResourceParts[1]
	return
}

// func expandMapOfLists(in map[string]interface{}) map[string][]string {
// 	m := map[string][]string{}
// 	for s := range in {
// 		log.Printf("[INFO] ------------------- s -------------  %s", s)
// 		i := strings.LastIndex(s, ".")
// 		log.Printf("[INFO] ------------------- i -------------  %s", s)
// 		first := s[0:i]
// 		log.Printf("[INFO] ------------------- first -------------  %s", first)
// 		last := s[i+1:]
// 		log.Printf("[INFO] ------------------- last -------------  %s", last)
// 		if last != "#" {
// 			log.Printf("[INFO] ------------------- last -------------  %s", m[first])
// 			m[first] = append(m[first], in[s].(string))
// 			log.Printf("[INFO] ------------------- last -------------  %s", in[s].(string))
// 		}
// 	}
// 	return m
// }

// Convert map[string][]string to map[string]string
func convertToStringMap(input map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range input {
		result[key] = strings.Join(value, ",") // Join slice into a single string
	}
	return result
}

// Convert map[string]string to map[string][]string
func convertToSliceMap(input map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	for key, value := range input {
		result[key] = strings.Split(value.(string), ",") // Split string back into slice
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

//endregion
