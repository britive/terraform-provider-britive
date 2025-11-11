package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceResourceManagerProfile - Terraform Resource for Profile Management
type ResourceResourceManagerProfile struct {
	Resource     *schema.Resource
	helper       *ResourceResourceManagerProfileHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceResourceManagerProfileHelper - Helper for Profile Management Resource
type ResourceResourceManagerProfileHelper struct{}

func NewResourceResourceManagerProfileHelper() *ResourceResourceManagerProfileHelper {
	return &ResourceResourceManagerProfileHelper{}
}

func NewResourceResourceManagerProfile(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResourceManagerProfile {
	rrmp := &ResourceResourceManagerProfile{
		helper:       NewResourceResourceManagerProfileHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rrmp.Resource = &schema.Resource{
		CreateContext: rrmp.resourceCreate,
		UpdateContext: rrmp.resourceUpdate,
		ReadContext:   rrmp.resourceRead,
		DeleteContext: rrmp.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rrmp.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the britive resource manager profile",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of britive resource manager profile",
			},
			"expiration_duration": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Expiration duration of resource manager profile",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of resource manager profile",
			},
			"delegation_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable or disable delegation",
			},
			"associations": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Resource manager profile associations",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Resource label name for association",
						},
						"values": {
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Description: "Values of resource label for association",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"resource_label_color_map": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Resource label color map",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label_key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "name of the resource label",
						},
						"color_code": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "color code of resource label",
						},
					},
				},
			},
		},
	}
	return rrmp
}

func (rrmp *ResourceResourceManagerProfile) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	resourceManagerProfile := &britive.ResourceManagerProfile{}

	err := rrmp.helper.mapResourceToModel(d, resourceManagerProfile)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating resource_manager_profile Resource : %#v", resourceManagerProfile)
	resourceManagerProfile, err = c.CreateUpdateResourceManagerProfile(*resourceManagerProfile, false)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("Resource manager profile"))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	profileID := resourceManagerProfile.ProfileId

	log.Printf("[INFO] Adding associations to resource_manager_profile")
	_, err = c.CreateUpdateResourceManagerProfileAssociations(*resourceManagerProfile)
	if errors.Is(err, britive.ErrNotFound) {
		diags = append(diags, diag.FromErr(errs.NewNotFoundErrorf("Resource manager profile"))...)
	}
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if len(diags) > 0 {
		log.Printf("[WARN] Rolling back profile creation due to error: %v", diags)
		delErr := c.DeleteResourceManagerProfile(profileID)
		if delErr != nil {
			log.Printf("[ERROR] Failed to delete profile during rollback: %v", delErr)
		} else {
			log.Printf("[INFO] Rolled back profile creation")
		}
		return diags
	}

	d.SetId(rrmp.helper.generateUniqueId(profileID))

	log.Printf("[INFO] Added Associations to Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	return rrmp.resourceRead(ctx, d, m)
}

func (rrmp *ResourceResourceManagerProfile) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	err, profileId := rrmp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading Resource_Manager_Profile Resource of ID : %s", profileId)

	resourceManagerProfile, err := c.GetResourceManagerProfile(profileId)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("resource-manager profile"))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading Associations from Resource_Manager_Profile Resource : %#v", resourceManagerProfile)
	associations, err := c.GetResourceManagerProfileAssociations(profileId)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("resource-manager profile"))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	resourceManagerProfile.Associations = associations.Associations
	resourceManagerProfile.ResourceLabelColorMap = associations.ResourceLabelColorMap

	err = rrmp.helper.getAndMapModelToResource(d, *resourceManagerProfile)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Found Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

	return diags
}

func (rrmp *ResourceResourceManagerProfile) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("expiration_duration") || d.HasChange("associations") || d.HasChange("delegation_enabled") {
		resourceManagerProfile := &britive.ResourceManagerProfile{}
		err := rrmp.helper.mapResourceToModel(d, resourceManagerProfile)
		if err != nil {
			return diag.FromErr(err)
		}

		err, profileId := rrmp.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		resourceManagerProfile.ProfileId = profileId

		log.Printf("[INFO] Updating Resource_Manager_Profile Resource : %#v", resourceManagerProfile)

		_, err = c.CreateUpdateResourceManagerProfile(*resourceManagerProfile, true)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("resource-manager profile"))
		}
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating associations to resource_manager_profile")
		_, err = c.CreateUpdateResourceManagerProfileAssociations(*resourceManagerProfile)
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("resource-manager profile"))
		}
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updated Resource_Manager_Profile Resource")
	}

	return rrmp.resourceRead(ctx, d, m)
}

func (rrmp *ResourceResourceManagerProfile) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	err, profileId := rrmp.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting Resource_manager_Profile Resource of ID : %s", profileId)

	err = c.DeleteResourceManagerProfile(profileId)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("resource-manager profile", profileId))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	log.Printf("[INFO] Deleted Resource_Manager_Profile Resource of ID : %s", profileId)

	return diags
}

func (rrmp *ResourceResourceManagerProfile) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rrmp.importHelper.ParseImportID([]string{"resource-manager/profile/(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	profileId := d.Id()
	log.Printf("[INFO] Importing Resource_Manager_Profile Resource : %s", profileId)

	resp, err := c.GetResourceManagerProfile(profileId)
	if err != nil {
		return nil, err
	}

	d.SetId(rrmp.helper.generateUniqueId(resp.ProfileId))
	log.Printf("[INFO] Imported Resource_Manager_Profile Resource : %#v", resp)
	return []*schema.ResourceData{d}, nil
}

func (helper *ResourceResourceManagerProfileHelper) mapResourceToModel(d *schema.ResourceData, resourceManagerProfile *britive.ResourceManagerProfile) error {
	resourceManagerProfile.Name = d.Get("name").(string)
	if v, ok := d.GetOk("description"); ok {
		resourceManagerProfile.Description = v.(string)
	}
	if delegationEnabled, ok := d.GetOk("delegation_enabled"); ok {
		resourceManagerProfile.DelegationEnabled = delegationEnabled.(bool)
	}
	resourceManagerProfile.ExpirationDuration = d.Get("expiration_duration").(int)

	rawAssociations := d.Get("associations").(*schema.Set)
	associationMap := make(map[string][]string)
	for _, association := range rawAssociations.List() {
		userAssociationMap := association.(map[string]interface{})
		resourceLabelName := userAssociationMap["label_key"].(string)
		rawValues := userAssociationMap["values"].(*schema.Set).List()
		valuesList := make([]string, len(rawValues))
		for i, val := range rawValues {
			valuesList[i] = val.(string)
		}
		associationMap[resourceLabelName] = valuesList
	}
	resourceManagerProfile.Associations = associationMap

	return nil
}

func (helper *ResourceResourceManagerProfileHelper) getAndMapModelToResource(d *schema.ResourceData, resourceManagerProfile britive.ResourceManagerProfile) error {
	d.Set("name", resourceManagerProfile.Name)
	d.Set("description", resourceManagerProfile.Description)
	d.Set("expiration_duration", resourceManagerProfile.ExpirationDuration)
	d.Set("status", resourceManagerProfile.Status)
	if err := d.Set("delegation_enabled", resourceManagerProfile.DelegationEnabled); err != nil {
		return err
	}

	var associationMapList []map[string]interface{}
	for name, values := range resourceManagerProfile.Associations {
		associationMap := map[string]interface{}{
			"label_key": name,
			"values":    values,
		}
		associationMapList = append(associationMapList, associationMap)
	}
	d.Set("associations", associationMapList)

	var colorCodeMapList []map[string]interface{}
	for name, color := range resourceManagerProfile.ResourceLabelColorMap {
		colorCodeMap := map[string]interface{}{
			"label_key":  name,
			"color_code": color,
		}
		colorCodeMapList = append(colorCodeMapList, colorCodeMap)
	}
	d.Set("resource_label_color_map", colorCodeMapList)

	return nil
}

func (helper *ResourceResourceManagerProfileHelper) generateUniqueId(profileId string) string {
	return fmt.Sprintf("resource-manager/profile/%s", profileId)
}

func (helper *ResourceResourceManagerProfileHelper) parseUniqueID(rawId string) (error, string) {
	rawArr := strings.Split(rawId, "/")
	if len(rawArr) != 3 {
		return errs.NewInvalidResourceIDError("resource-manager profile", rawId), ""
	}
	return nil, rawArr[len(rawArr)-1]
}
