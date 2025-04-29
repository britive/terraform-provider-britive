package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/errs"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// SessionAttributeType - godoc
type SessionAttributeType string

const (
	//SessionAttributeTypeStatic - godoc
	SessionAttributeTypeStatic SessionAttributeType = "Static"
	//SessionAttributeTypeIdentity - godoc
	SessionAttributeTypeIdentity SessionAttributeType = "Identity"
)

// ResourceProfileSessionAttribute - Terraform Resource for Profile Session Attribute
type ResourceProfileSessionAttribute struct {
	Resource     *schema.Resource
	helper       *ResourceProfileSessionAttributeHelper
	importHelper *imports.ImportHelper
}

// NewResourceProfileSessionAttribute - Initialization of new profile session attribute resource
func NewResourceProfileSessionAttribute(importHelper *imports.ImportHelper) *ResourceProfileSessionAttribute {
	rpt := &ResourceProfileSessionAttribute{
		helper:       NewResourceProfileSessionAttributeHelper(),
		importHelper: importHelper,
	}
	rpt.Resource = &schema.Resource{
		CreateContext: rpt.resourceCreate,
		ReadContext:   rpt.resourceRead,
		UpdateContext: rpt.resourceUpdate,
		DeleteContext: rpt.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rpt.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"app_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The application name of the application, profile is assciated with",
			},
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"profile_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the profile",
			},
			"attribute_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The attribute name associate with the profile",
			},
			"attribute_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The type of attribute associate with the profile, should be one of [Static, Identity]",
				Default:      "Identity",
				ValidateFunc: validation.StringInSlice([]string{"Static", "Identity"}, false),
			},
			"attribute_value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The attribute value associate with the profile",
			},
			"mapping_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The attribute mapping name associate with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"transitive": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The attribute transitive associate with the profile",
			},
		},
	}
	return rpt
}

//region Profile Tag Resource Context Operations

func (rpt *ResourceProfileSessionAttribute) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	profileID := d.Get("profile_id").(string)
	sessionAttribute, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new profile session attribute: %#v", *sessionAttribute)

	pt, err := c.CreateProfileSessionAttribute(profileID, *sessionAttribute)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new profile session attribute: %#v", *pt)

	d.SetId(rpt.helper.generateUniqueID(profileID, pt.ID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileSessionAttribute) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	err := rpt.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rpt *ResourceProfileSessionAttribute) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("mapping_name") &&
		!d.HasChange("attribute_value") &&
		!d.HasChange("transitive") {
		return nil
	}

	c := m.(*britive.Client)
	profileID, sessionAttributeID, err := rpt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	sessionAttribute, err := rpt.helper.getAndMapResourceToModel(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	sessionAttribute.ID = sessionAttributeID

	log.Printf("[INFO] Updating profile session attribute: %#v", *sessionAttribute)

	upt, err := c.UpdateProfileSessionAttribute(profileID, *sessionAttribute)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted Updated profile session attribute: %#v", upt)

	d.SetId(rpt.helper.generateUniqueID(profileID, sessionAttributeID))

	return rpt.resourceRead(ctx, d, m)
}

func (rpt *ResourceProfileSessionAttribute) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, sessionAttributeID, err := rpt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting profile session attribute: %s/%s", profileID, sessionAttributeID)

	err = c.DeleteProfileSessionAttribute(profileID, sessionAttributeID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleted profile session attribute: %s/%s", profileID, sessionAttributeID)

	d.SetId("")

	return diags
}

func (rpt *ResourceProfileSessionAttribute) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	if err := rpt.importHelper.ParseImportID([]string{"apps/(?P<app_name>[^/]+)/paps/(?P<profile_name>[^/]+)/session-attributes/type/(?P<attribute_type>[^/]+)/mapping-name/(?P<mapping_name>[^/]+)", "(?P<app_name>[^/]+)/(?P<profile_name>[^/]+)/(?P<attribute_type>[^/]+)/(?P<mapping_name>[^/]+)"}, d); err != nil {
		return nil, err
	}
	appName := d.Get("app_name").(string)
	profileName := d.Get("profile_name").(string)
	attributeType := d.Get("attribute_type").(string)
	mappingName := d.Get("mapping_name").(string)
	if strings.TrimSpace(appName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("app_name")
	}
	if strings.TrimSpace(profileName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_name")
	}
	if strings.TrimSpace(attributeType) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("attribute_type")
	}
	if strings.TrimSpace(mappingName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("mapping_name")
	}

	log.Printf("[INFO] Importing profile session attribute: %s/%s/%s/%s", appName, profileName, attributeType, mappingName)

	app, err := c.GetApplicationByName(appName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("application %s", appName)
	}
	if err != nil {
		return nil, err
	}
	profile, err := c.GetProfileByName(app.AppContainerID, profileName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("profile %s", profileName)
	}
	if err != nil {
		return nil, err
	}

	sessionAttribute, err := c.GetProfileSessionAttributeByTypeAndMappingName(profile.ProfileID, attributeType, mappingName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("session attribute with type %s and mapping name %s", attributeType, mappingName)
	}
	if err != nil {
		return nil, err
	}

	d.SetId(rpt.helper.generateUniqueID(profile.ProfileID, sessionAttribute.ID))
	d.Set("app_name", "")
	d.Set("profile_name", "")

	err = rpt.helper.getAndMapModelToResource(d, m)
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Imported profile session attribute: %s/%s/%s/%s", appName, profileName, attributeType, mappingName)
	return []*schema.ResourceData{d}, nil
}

//endregion

// ResourceProfileSessionAttributeHelper - Terraform Resource for Profile Tag Helper
type ResourceProfileSessionAttributeHelper struct {
}

// NewResourceProfileSessionAttributeHelper - Initialization of new profile session attribute resource helper
func NewResourceProfileSessionAttributeHelper() *ResourceProfileSessionAttributeHelper {
	return &ResourceProfileSessionAttributeHelper{}
}

//region Profile Tag Helper functions

func (resourceProfileSessionAttributeHelper *ResourceProfileSessionAttributeHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)
	profileID, sessionAttributeID, err := resourceProfileSessionAttributeHelper.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading profile session attribute: %s/%s", profileID, sessionAttributeID)

	pt, err := c.GetProfileSessionAttribute(profileID, sessionAttributeID)
	if errors.Is(err, britive.ErrNotFound) {
		return errs.NewNotFoundErrorf("session attribute %s in profile %s", sessionAttributeID, profileID)
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received profile session attribute: %#v", pt)

	var attributeName string
	var attributeValue string
	if strings.EqualFold(pt.SessionAttributeType, string(SessionAttributeTypeIdentity)) || pt.AttributeSchemaID != "" {
		log.Printf("[INFO] Reading attribute: %s/%s", profileID, sessionAttributeID)

		attribute, err := c.GetAttribute(pt.AttributeSchemaID)
		if errors.Is(err, britive.ErrNotFound) {
			return errs.NewNotFoundErrorf("attribute %s", pt.AttributeSchemaID)
		}
		if err != nil {
			return err
		}

		log.Printf("[INFO] Received attribute: %#v", attribute)

		attributeName = attribute.Name
	} else {
		attributeValue = pt.AttributeValue
	}
	d.Set("profile_id", profileID)
	d.Set("attribute_name", attributeName)
	d.Set("attribute_type", pt.SessionAttributeType)
	d.Set("attribute_value", attributeValue)
	d.Set("mapping_name", pt.MappingName)
	d.Set("transitive", pt.Transitive)

	return nil
}

func (resourceProfileSessionAttributeHelper *ResourceProfileSessionAttributeHelper) getAndMapResourceToModel(d *schema.ResourceData, m interface{}) (*britive.SessionAttribute, error) {
	c := m.(*britive.Client)
	attributeName := d.Get("attribute_name").(string)
	mappingName := d.Get("mapping_name").(string)
	attributeType := d.Get("attribute_type").(string)
	attributeValuePassed := d.Get("attribute_value").(string)
	transitive := d.Get("transitive").(bool)
	if attributeType == "" {
		attributeType = string(SessionAttributeTypeIdentity)
	}
	var attributeSchemaId string
	var attributeValue string
	if strings.EqualFold(attributeType, string(SessionAttributeTypeIdentity)) {
		if attributeName == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("attribute_name")
		}
		if attributeValuePassed != "" {
			return nil, fmt.Errorf("expected attribute_value should be empty when attribute_type is %s", attributeType)
		}
		attribute, err := c.GetAttributeByName(attributeName)
		if errors.Is(err, britive.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("session attribute %s", attributeName)
		}
		if err != nil {
			return nil, err
		}
		attributeSchemaId = attribute.ID
	} else {
		if attributeValuePassed == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("attribute_value")
		}
		if attributeName != "" {
			return nil, fmt.Errorf("expected attribute_name should be empty when attribute_type is %s", attributeType)
		}
		attributeValue = attributeValuePassed
	}
	profileSessionAttribute := britive.SessionAttribute{
		AttributeSchemaID:    attributeSchemaId,
		MappingName:          mappingName,
		Transitive:           transitive,
		SessionAttributeType: attributeType,
		AttributeValue:       attributeValue,
	}
	return &profileSessionAttribute, nil
}

func (resourceProfileSessionAttributeHelper *ResourceProfileSessionAttributeHelper) generateUniqueID(profileID string, attributeID string) string {
	return fmt.Sprintf("paps/%s/session-attributes/%s", profileID, attributeID)
}

func (resourceProfileSessionAttributeHelper *ResourceProfileSessionAttributeHelper) parseUniqueID(ID string) (profileID string, attributeID string, err error) {
	profileTagParts := strings.Split(ID, "/")
	if len(profileTagParts) < 4 {
		err = errs.NewInvalidResourceIDError("profile session attribute", ID)
		return
	}
	profileID = profileTagParts[1]
	attributeID = profileTagParts[3]
	return
}

//endregion
