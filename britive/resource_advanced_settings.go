package britive

import (
	"context"
	"fmt"
	"log"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type ResourceAdvancedSettings struct {
	Resource     *schema.Resource
	helper       *ResourceApplicationSettingsHelper
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
		ReadContext:   rst.resourceRead,
		DeleteContext: rst.resourceDelete,
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Britive resource id",
				ForceNew:    true,
			},
			"resource_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Britive resource type",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"APPLICATION", "PROFILE", "PROFILE POLICY", "RESOURCE MANAGER PROFILE POLICY"}, true),
			},
			"justification_setting": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Resource's Justification Settings",
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"is_justification_required": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Resource justification",
						},
						"justification_regex": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Resource justification Regular Expression",
						},
					},
				},
			},
			"itsm_setting": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Resource ITSM Setting",
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"connection_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "ITSM Connection id",
						},
						"connection_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "ITSM Connection type",
						},
						"is_comment_required": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "itsm comment",
						},
						"is_itsm_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "itsm comment",
						},
						"itsm_filter_criteria": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "filters",
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

	applicationSettings := britive.AdvancedSettings{}
	err := rst.helper.mapApplicationSettingResourceToModel(d, m, &applicationSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	appSettingResponse, err := c.CreateApplicationSettings(applicationSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("========== created settings %v", appSettingResponse)

	d.SetId(appSettingResponse.Settings[0].ID)

	return diags
}

func (rst *ResourceAdvancedSettings) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// For now, assume resource always exists.
	return nil
}

func (rst *ResourceAdvancedSettings) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// For now, assume resource always exists.
	return nil
}

func (rrst *ResourceApplicationSettingsHelper) mapApplicationSettingResourceToModel(d *schema.ResourceData, m interface{}, applicationSettings *britive.AdvancedSettings) error {
	resourceId := d.Get("resource_id").(string)
	resourceType := d.Get("resource_type").(string)

	// Handle justification settings
	if justificationRaw, ok := d.GetOk("justification_setting"); ok {
		justificationList := justificationRaw.(*schema.Set).List()
		if len(justificationList) != 1 {
			return fmt.Errorf("invalid justification settings: must contain exactly one element")
		}

		userJustificationSetting := justificationList[0].(map[string]interface{})
		justificationSetting := britive.Setting{
			SettingsType: "JUSTIFICATION",
			EntityID:     resourceId,
			EntityType:   resourceType,
		}

		if val, ok := userJustificationSetting["is_justification_required"].(bool); ok {
			justificationSetting.IsJustificationRequired = val
		}

		if val, ok := userJustificationSetting["justification_regex"].(string); ok {
			justificationSetting.JustificationRegex = val
		}

		applicationSettings.Settings = append(applicationSettings.Settings, justificationSetting)
	}

	// Handle ITSM settings
	if itsmRaw, ok := d.GetOk("itsm_setting"); ok {
		itsmList := itsmRaw.(*schema.Set).List()
		if len(itsmList) != 1 {
			return fmt.Errorf("invalid ITSM settings: must contain exactly one element")
		}

		userItsmSetting := itsmList[0].(map[string]interface{})
		itsmSetting := britive.Setting{
			SettingsType: "ITSM",
			EntityID:     resourceId,
			EntityType:   resourceType,
		}

		if val, ok := userItsmSetting["connection_id"].(string); ok {
			itsmSetting.ConnectionID = val
		}
		if val, ok := userItsmSetting["connection_type"].(string); ok {
			itsmSetting.ConnectionType = val
		}
		if val, ok := userItsmSetting["is_comment_required"].(bool); ok {
			itsmSetting.IsCommentRequired = val
		}
		if val, ok := userItsmSetting["is_itsm_enabled"].(bool); ok {
			itsmSetting.IsITSMEnabled = val
		}

		applicationSettings.Settings = append(applicationSettings.Settings, itsmSetting)
	}

	return nil
}

type ResourceApplicationSettingsHelper struct {
}

func NewResourceAdvancedSettingsHelper() *ResourceApplicationSettingsHelper {
	return &ResourceApplicationSettingsHelper{}
}
