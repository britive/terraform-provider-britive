package britive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type ResourceAdvancedSettings struct {
	Resource     *schema.Resource
	helper       *ResourceAdvancedSettingsHelper
	validation   *Validation
	importHelper *ImportHelper
}

func NewResourceAdvancedSettings(v *Validation, importHelper *ImportHelper) *ResourceAdvancedSettings {
	rst := &ResourceAdvancedSettings{
		helper:       NewResourceAdvancedSettingsHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rst.Resource = &schema.Resource{
		CreateContext: rst.resourceCreate,
		UpdateContext: rst.resourceUpdate,
		ReadContext:   rst.resourceRead,
		DeleteContext: rst.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rst.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Britive resource id",
			},
			"resource_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Britive resource type",
				ValidateFunc: validation.StringInSlice([]string{"APPLICATION", "PROFILE", "PROFILE_POLICY", "RESOURCE_MANAGER_PROFILE", "RESOURCE_MANAGER_PROFILE_POLICY"}, true),
			},
			"justification_settings": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Resource's Justification Settings",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"justification_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Justification Setting ID",
						},
						"is_justification_required": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Resource justification",
						},
						"justification_regex": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Resource justification Regular Expression",
						},
					},
				},
			},
			"itsm_setting": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Resource ITSM Setting",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"itsm_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ITSM Setting ID",
						},
						"connection_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ITSM Connection id",
						},
						"connection_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ITSM Connection type",
						},
						"is_itsm_enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "itsm comment",
						},
						"itsm_filter_criteria": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "filters",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"supported_ticket_type": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "supported ticket type",
									},
									"filter": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "filter",
										ValidateFunc: func(i interface{}, s string) (warns []string, errs []error) {
											str := i.(string)
											var js interface{}
											if err := json.Unmarshal([]byte(str), &js); err != nil {
												errs = append(errs, fmt.Errorf("%q contains invalid JSON: %s", s, err))
											}
											return
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return rst
}

func (rst *ResourceAdvancedSettings) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	resourceId := d.Get("resource_id").(string)
	resourceType := d.Get("resource_type").(string)
	resourceType = strings.ToUpper(resourceType)

	if resourceId == "" || resourceType == "" {
		return diag.FromErr(fmt.Errorf("ResourceID or ResourceType Cannot be empty."))
	}

	advancedSettings := britive.AdvancedSettings{}
	err := rst.helper.mapAdvancedSettingResourceToModel(d, m, &advancedSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new advanced settings %#v", advancedSettings)

	isUpdate := false
	if d.Id() != "" {
		isUpdate = true
	}

	if err := c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, isUpdate); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new advanced settings: %#v", advancedSettings)

	settingId := rst.helper.generateUniqueID(resourceId, resourceType)

	d.SetId(settingId)

	rst.resourceRead(ctx, d, m)

	log.Printf("[INFO] Updated state after advaned settings submission: %#v", advancedSettings)

	return diags
}

func (rst *ResourceAdvancedSettings) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	advancedSettings := britive.AdvancedSettings{}

	log.Printf("[INFO] Mapping resource to model for update")

	err := rst.helper.mapAdvancedSettingResourceToModel(d, m, &advancedSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceId := d.Get("resource_id").(string)
	resourceType := d.Get("resource_type").(string)
	resourceType = strings.ToUpper(resourceType)

	if resourceId == "" || resourceType == "" {
		return diag.FromErr(fmt.Errorf("ResourceID or ResourceType Cannot be empty."))
	}

	isUpdate := false
	if d.Id() != "" {
		isUpdate = true
	}

	log.Printf("[INFO] Updating advanced settings: %#v", advancedSettings)

	if err := c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, isUpdate); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Updated advanced settings: %#v", advancedSettings)

	rst.resourceRead(ctx, d, m)

	log.Printf("[INFO] Updated state after advanced settings: %#v", advancedSettings)
	return diags
}

func (rst *ResourceAdvancedSettings) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	err := rst.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rst *ResourceAdvancedSettings) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics
	advancedSettings := britive.AdvancedSettings{}

	resourceId := d.Get("resource_id").(string)
	resourceType := d.Get("resource_type").(string)
	resourceType = strings.ToUpper(resourceType)

	if resourceId == "" || resourceType == "" {
		return diag.FromErr(fmt.Errorf("ResourceID or ResourceType Cannot be empty."))
	}

	isUpdate := false
	if d.Id() != "" {
		isUpdate = true
	}

	log.Printf("[INFO] Deleting advanced settings of %s", resourceId)

	if err := c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, isUpdate); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	log.Printf("[INFO] Advanced settings %v deleted", resourceId)
	return diags
}

func (rrst *ResourceAdvancedSettingsHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	resourceID, resourceType := rrst.parseUniqueID(d.Id())
	log.Printf("[INFO] Reading advanced settings %s", resourceID)
	resourceType = strings.ToUpper(resourceType)

	rawResourceID := d.Get("resource_id")
	resourceID = rawResourceID.(string)

	advancedSettings, err := c.GetAdvancedSettings(resourceID, resourceType)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading application %#v", advancedSettings)

	var rawJustificationSetting britive.Setting
	var rawItsmSetting britive.Setting

	for _, rawSetting := range advancedSettings.Settings {
		if rawSetting.SettingsType == "JUSTIFICATION" {
			rawJustificationSetting = rawSetting
		} else if rawSetting.SettingsType == "ITSM" {
			rawItsmSetting = rawSetting
		}
	}

	if rawJustificationSetting.ID != "" && !(rawJustificationSetting.IsInherited != nil && *rawJustificationSetting.IsInherited == true) {
		justificationSetting := []map[string]interface{}{
			{
				"is_justification_required": rawJustificationSetting.IsJustificationRequired,
				"justification_regex":       rawJustificationSetting.JustificationRegex,
				"justification_id":          rawJustificationSetting.ID,
			},
		}
		if err := d.Set("justification_settings", justificationSetting); err != nil {
			return err
		}
	} else {
		if err := d.Set("justification_settings", nil); err != nil {
			return err
		}
	}

	if rawItsmSetting.ID != "" && !(rawItsmSetting.IsInherited != nil && *rawItsmSetting.IsInherited == true) {
		itsmFilterCriteria := []map[string]interface{}{}
		for _, criteria := range rawItsmSetting.ItsmFilterCriterias {
			filterStr := ""
			if criteria.Filter != nil {
				bytes, err := json.Marshal(criteria.Filter)
				if err != nil {
					return fmt.Errorf("fInvalid ITSM filter criteria filter: %w. Filter should be json decodable.", err)
				}
				filterStr = string(bytes)
			}
			itsmFilterCriteria = append(itsmFilterCriteria, map[string]interface{}{
				"supported_ticket_type": criteria.SupportedTicketType,
				"filter":                filterStr,
			})
		}

		itsmSetting := []map[string]interface{}{
			{
				"connection_id":        rawItsmSetting.ConnectionID,
				"connection_type":      rawItsmSetting.ConnectionType,
				"is_itsm_enabled":      rawItsmSetting.IsITSMEnabled,
				"itsm_filter_criteria": itsmFilterCriteria,
				"itsm_id":              rawItsmSetting.ID,
			},
		}

		if err := d.Set("itsm_setting", itsmSetting); err != nil {
			return err
		}

	} else {
		if err := d.Set("itsm_setting", nil); err != nil {
			return err
		}
	}

	log.Printf("[INFO] Updated state of advanced settings %#v", advancedSettings)

	return nil
}

func (rrst *ResourceAdvancedSettingsHelper) mapAdvancedSettingResourceToModel(d *schema.ResourceData, m interface{}, advancedSettings *britive.AdvancedSettings) error {
	// c := m.(*britive.Client)
	isInherited := false
	resourceId := d.Get("resource_id").(string)
	resourceType := d.Get("resource_type").(string)
	resourceTypeArr := strings.Split(resourceType, "_")
	resourceType = resourceTypeArr[len(resourceTypeArr)-1]
	resourceType = strings.ToUpper(resourceType)

	resourceIDArr := strings.Split(resourceId, "/")
	if len(resourceIDArr) > 1 {
		resourceId = resourceIDArr[len(resourceIDArr)-1]
	}

	// Handle justification settings
	if justificationRaw, ok := d.GetOk("justification_settings"); ok {
		justificationList := justificationRaw.([]interface{})
		if len(justificationList) != 1 {
			return fmt.Errorf("Invalid justification settings: must contain exactly one element")
		}

		userJustificationSetting := justificationList[0].(map[string]interface{})
		justificationSetting := britive.Setting{
			SettingsType: "JUSTIFICATION",
			EntityID:     resourceId,
			EntityType:   resourceType,
			IsInherited:  &isInherited,
		}

		if val, ok := userJustificationSetting["justification_id"].(string); ok {
			justificationSetting.ID = val
		}

		if val, ok := userJustificationSetting["is_justification_required"].(bool); ok {
			justificationSetting.IsJustificationRequired = &val
		}

		if val, ok := userJustificationSetting["justification_regex"].(string); ok {
			justificationSetting.JustificationRegex = val
		}

		advancedSettings.Settings = append(advancedSettings.Settings, justificationSetting)
	}

	// Handle ITSM settings
	if itsmRaw, ok := d.GetOk("itsm_setting"); ok {
		itsmList := itsmRaw.([]interface{})
		if len(itsmList) != 1 {
			return fmt.Errorf("Invalid ITSM settings: must contain exactly one element")
		}

		userItsmSetting := itsmList[0].(map[string]interface{})
		itsmSetting := britive.Setting{
			SettingsType: "ITSM",
			EntityID:     resourceId,
			EntityType:   resourceType,
			IsInherited:  &isInherited,
		}

		if val, ok := userItsmSetting["itsm_id"].(string); ok {
			itsmSetting.ID = val
		}

		if val, ok := userItsmSetting["connection_id"].(string); ok {
			itsmSetting.ConnectionID = val
		}
		if val, ok := userItsmSetting["connection_type"].(string); ok {
			itsmSetting.ConnectionType = val
		}

		if val, ok := userItsmSetting["is_itsm_enabled"].(bool); ok {
			itsmSetting.IsITSMEnabled = &val
		}
		if rawSet, ok := userItsmSetting["itsm_filter_criteria"].(*schema.Set); ok {
			values := rawSet.List()
			for _, item := range values {
				val := item.(map[string]interface{})

				var js map[string]interface{}
				if err := json.Unmarshal([]byte(val["filter"].(string)), &js); err != nil {
					return fmt.Errorf("Invalid JSON format in itsm_filter_criteria 'filter': %v", err)
				}

				itsmFilter := britive.ItsmFilterCriteria{
					SupportedTicketType: val["supported_ticket_type"].(string),
					Filter:              make(map[string]interface{}),
				}

				for k, v := range js {
					itsmFilter.Filter[k] = v
				}
				itsmSetting.ItsmFilterCriterias = append(itsmSetting.ItsmFilterCriterias, itsmFilter)
			}
		}

		advancedSettings.Settings = append(advancedSettings.Settings, itsmSetting)
	}

	return nil
}

type ResourceAdvancedSettingsHelper struct {
}

func NewResourceAdvancedSettingsHelper() *ResourceAdvancedSettingsHelper {
	return &ResourceAdvancedSettingsHelper{}
}

func (rrst *ResourceAdvancedSettingsHelper) generateUniqueID(resourceID, resourceType string) string {
	resourceType = strings.ToUpper(resourceType)
	resourceArr := strings.Split(resourceID, "/")
	if len(resourceArr) > 1 {
		return resourceType + "/" + resourceArr[len(resourceArr)-1] + "/advanced-settings"
	}
	generatedID := resourceType + "/" + resourceID + "/advanced-settings"

	log.Printf("[INFO] Generated advanced settings ID: %s", generatedID)

	return generatedID
}

func (rrst *ResourceAdvancedSettingsHelper) parseUniqueID(ID string) (string, string) {
	arr := strings.Split(ID, "/")
	resourceId := arr[1]
	resourceType := arr[0]
	resourceType = strings.ToUpper(resourceType)
	return resourceId, resourceType
}

func (rst *ResourceAdvancedSettings) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	importID := d.Id()

	importArr := strings.Split(importID, "/")
	resourceType := importArr[len(importArr)-1]
	resourceType = strings.ToUpper(resourceType)

	importArr = importArr[:len(importArr)-1]
	resourceID := strings.Join(importArr, "/")

	resource := resourceID

	log.Printf("[INFO] Importing advanced settings, %s", resourceID)

	advancedSettings, err := c.GetAdvancedSettings(resourceID, resourceType)
	if err != nil {
		return nil, err
	}

	if len(importArr) > 1 {
		resourceID = importArr[len(importArr)-1]
	} else {
		resourceID = importArr[0]
	}

	d.SetId(rst.helper.generateUniqueID(resourceID, resourceType))
	d.Set("resource_id", resource)
	d.Set("resource_type", resourceType)

	log.Printf("[INFO] Imported advanced settings: %s, advanced settin: %#v", resourceID, advancedSettings)

	return []*schema.ResourceData{d}, nil
}
