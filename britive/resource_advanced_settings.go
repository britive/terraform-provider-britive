package britive

import (
	"context"
	"encoding/json"
	"errors"
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
				ForceNew:     true,
				Description:  "Britive resource type",
				ValidateFunc: validation.StringInSlice([]string{"APPLICATION", "PROFILE", "PROFILE_POLICY"}, true),
			},
			"justification_settings": {
				Type:        schema.TypeList,
				MaxItems:    1,
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
							Optional:    true,
							Description: "Resource justification Regular Expression",
							Default:     "",
						},
					},
				},
			},
			"itsm": {
				Type:        schema.TypeList,
				MaxItems:    1,
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
												errs = append(errs, err)
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
			"im": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Resource IM Setting",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"im_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IM Setting ID",
						},
						"connection_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "IM Connection id",
						},
						"connection_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "IM Connection type",
						},
						"is_auto_approval_enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "IM auto approval toggle",
						},
						"escalation_policies": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "IM Escalation Policies",
							Elem: &schema.Schema{
								Type: schema.TypeString,
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
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_id"))
	}
	if resourceType == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_type"))
	}

	advancedSettings := britive.AdvancedSettings{}
	err := rst.helper.mapAdvancedSettingResourceToModel(d, m, &advancedSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Adding new advanced settings %#v", advancedSettings)

	advancedSettingsCheck, err := c.GetAdvancedSettings(resourceId, resourceType)
	if errors.Is(err, britive.ErrNotFound) {
		err = NewNotFoundErrorf("advanced settings of %s", resourceId)
	} else if errors.Is(err, britive.ErrNotSupported) {
		err = NewNotSupportedError(resourceType)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	isUpdate := false
	if len(advancedSettingsCheck.Settings) != 0 {
		isUpdate = true
	}

	err = c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, isUpdate)
	if errors.Is(err, britive.ErrNotFound) {
		err = NewNotFoundErrorf("advanced settings of %s", resourceId)
	} else if errors.Is(err, britive.ErrNotSupported) {
		err = NewNotSupportedError(resourceType)
	}
	if err != nil {
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
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_id"))
	}
	if resourceType == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_type"))
	}

	log.Printf("[INFO] Updating advanced settings: %#v", advancedSettings)

	err = c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, true)
	if errors.Is(err, britive.ErrNotFound) {
		err = NewNotFoundErrorf("advanced settings of %s", resourceId)
	} else if errors.Is(err, britive.ErrNotSupported) {
		err = NewNotSupportedError(resourceType)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	rst.resourceRead(ctx, d, m)

	log.Printf("[INFO] Updated state after advanced settings: %#v", advancedSettings)
	return diags
}

func (rst *ResourceAdvancedSettings) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	err := rst.helper.getAndMapModelToResource(d, m)
	if errors.Is(err, britive.ErrNotFound) {
		err = NewNotFoundErrorf("advanced settings")
	}
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
	resourceType = strings.ToLower(resourceType)

	if resourceId == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_id"))
	}
	if resourceType == "" {
		return diag.FromErr(NewNotFoundErrorf("resource_type"))
	}

	log.Printf("[INFO] Deleting advanced settings of %s", resourceId)

	err := c.CreateUpdateAdvancedSettings(resourceId, resourceType, advancedSettings, true)
	if errors.Is(err, britive.ErrNotFound) {
		err = NewNotFoundErrorf("advanced settings of %s", resourceId)
	} else if errors.Is(err, britive.ErrNotSupported) {
		err = NewNotSupportedError(resourceType)
	}
	if err != nil {
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

	rawResourceID := d.Get("resource_id")
	resourceID = rawResourceID.(string)

	advancedSettings, err := c.GetAdvancedSettings(resourceID, resourceType)
	if err != nil {
		return err
	}

	var rawJustificationSetting britive.Setting
	var rawItsmSetting britive.Setting
	var rawImSetting britive.Setting

	for _, rawSetting := range advancedSettings.Settings {
		if strings.EqualFold(rawSetting.SettingsType, "JUSTIFICATION") {
			rawJustificationSetting = rawSetting
		} else if strings.EqualFold(rawSetting.SettingsType, "ITSM") {
			rawItsmSetting = rawSetting
		} else if strings.EqualFold(rawSetting.SettingsType, "IM") {
			rawImSetting = rawSetting
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
					return err
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

		if err := d.Set("itsm", itsmSetting); err != nil {
			return err
		}

	} else {
		if err := d.Set("itsm", nil); err != nil {
			return err
		}
	}

	// Mapping IM Settings
	if rawImSetting.ID != "" && !(rawImSetting.IsInherited != nil && *rawImSetting.IsInherited == true) {
		var userConnType, connType string
		if imRaw, ok := d.GetOk("im"); ok {
			imList := imRaw.([]interface{})
			if len(imList) == 1 {
				im := imList[0].(map[string]interface{})
				userConnType = im["connection_type"].(string)
			}
		}
		if strings.EqualFold(rawImSetting.ConnectionType, userConnType) {
			connType = userConnType
		} else {
			connType = rawImSetting.ConnectionType
		}

		imSetting := []map[string]interface{}{
			{
				"connection_id":            rawImSetting.ConnectionID,
				"connection_type":          connType,
				"escalation_policies":      rawImSetting.EscalationPolicies,
				"is_auto_approval_enabled": rawImSetting.IsAutoApprovalEnabled,
				"im_id":                    rawImSetting.ID,
			},
		}
		if err := d.Set("im", imSetting); err != nil {
			return err
		}
	} else {
		if err := d.Set("im", nil); err != nil {
			return err
		}
	}

	log.Printf("[INFO] Updated state of advanced settings %#v", advancedSettings)

	return nil
}

func (rrst *ResourceAdvancedSettingsHelper) mapAdvancedSettingResourceToModel(d *schema.ResourceData, m interface{}, advancedSettings *britive.AdvancedSettings) error {
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
			return fmt.Errorf("Invalid: must contain exactly one justification setting")
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
	if itsmRaw, ok := d.GetOk("itsm"); ok {
		itsmList := itsmRaw.([]interface{})
		if len(itsmList) != 1 {
			return fmt.Errorf("Invalid: must contain exactly one ITSM setting")
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
					return err
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

	// Handle IM settings
	if imRaw, ok := d.GetOk("im"); ok {
		imList := imRaw.([]interface{})
		if len(imList) != 1 {
			return fmt.Errorf("Invalid: must contain exactly one IM setting")
		}

		userImSetting := imList[0].(map[string]interface{})
		imSetting := britive.Setting{
			SettingsType: "IM",
			EntityID:     resourceId,
			EntityType:   resourceType,
			IsInherited:  &isInherited,
		}

		if val, ok := userImSetting["im_id"].(string); ok {
			imSetting.ID = val
		}

		if val, ok := userImSetting["connection_id"].(string); ok {
			imSetting.ConnectionID = val
		}
		if val, ok := userImSetting["connection_type"].(string); ok {
			imSetting.ConnectionType = val
		}

		if val, ok := userImSetting["is_auto_approval_enabled"].(bool); ok {
			imSetting.IsAutoApprovalEnabled = &val
		}

		if rawSet, ok := userImSetting["escalation_policies"].(*schema.Set); ok {
			var policies []string
			for _, v := range rawSet.List() {
				if str, ok := v.(string); ok {
					policies = append(policies, str)
				}
			}
			imSetting.EscalationPolicies = policies
		}

		advancedSettings.Settings = append(advancedSettings.Settings, imSetting)
	}

	return nil
}

type ResourceAdvancedSettingsHelper struct {
}

func NewResourceAdvancedSettingsHelper() *ResourceAdvancedSettingsHelper {
	return &ResourceAdvancedSettingsHelper{}
}

func (rrst *ResourceAdvancedSettingsHelper) generateUniqueID(resourceID, resourceType string) string {
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
	return resourceId, resourceType
}

func (rst *ResourceAdvancedSettings) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	importID := d.Id()

	importArr := strings.Split(importID, "/")
	resourceType := importArr[len(importArr)-1]
	resourceType = strings.ToLower(resourceType)

	if !strings.EqualFold("application", resourceType) && !strings.EqualFold("profile", resourceType) && !strings.EqualFold("profile_policy", resourceType) {
		return nil, NewNotSupportedError(resourceType)
	}

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
