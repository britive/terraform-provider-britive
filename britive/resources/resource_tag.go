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

//ResourceTag - Terraform Resource for Tag
type ResourceTag struct {
	Resource     *schema.Resource
	helper       *ResourceTagHelper
	importHelper *imports.ImportHelper
}

//NewResourceTag - Initializes new tag resource
func NewResourceTag(importHelper *imports.ImportHelper) *ResourceTag {
	rt := &ResourceTag{
		helper:       NewResourceTagHelper(),
		importHelper: importHelper,
	}
	rt.Resource = &schema.Resource{
		CreateContext: rt.resourceCreate,
		ReadContext:   rt.resourceRead,
		UpdateContext: rt.resourceUpdate,
		DeleteContext: rt.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rt.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of Britive tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Britive tag",
			},
			"disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "To disable the Britive tag",
			},
			"identity_provider_id": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The unique identity of the identity provider associated with the Britive tag",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"external": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "The boolean attribute that indicates whether the tag is external or not",
			},
			"requestable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether the Britive tag is requestable",
			},
			"attributes": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The attributes of the Britive tag",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute_name": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The name of the attribute",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
						"attribute_value": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The value of the attribute",
							ValidateFunc: validation.StringIsNotWhiteSpace,
						},
					},
				},
			},
		},
	}
	return rt
}

//region Tag Resource Context Operations

func (rt *ResourceTag) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	err := rt.helper.validateForExternalTag(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	tag := britive.Tag{}
	tag.Name = d.Get("name").(string)
	tag.Description = d.Get("description").(string)
	if d.Get("disabled").(bool) {
		tag.Status = "Inactive"
	} else {
		tag.Status = "Active"
	}

	tag.UserTagIdentityProviders = []britive.UserTagIdentityProvider{
		{
			IdentityProvider: britive.IdentityProvider{
				ID: d.Get("identity_provider_id").(string),
			},
		},
	}
	log.Printf("[INFO] Creating new tag: %#v", tag)
	ut, err := c.CreateTag(tag)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new tag: %#v", ut)
	d.SetId(ut.ID)

	if rt.helper.hasAttributeConfig(d) {
		if diags := rt.helper.updateTagAttributes(c, ut.ID, d); diags != nil {
			return diags
		}
	}

	return rt.resourceRead(ctx, d, m)
}

func (rt *ResourceTag) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	err := rt.helper.validateForExternalTag(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	tagID := d.Id()

	log.Printf("[INFO] Reading tag %s", tagID)
	tag, err := c.GetTag(tagID)
	if errors.Is(err, britive.ErrNotFound) {
		return diag.FromErr(errs.NewNotFoundErrorf("tag %s", tagID))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received tag: %#v", tag)
	err = rt.helper.mapModelToResource(tag, d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rt *ResourceTag) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	err := rt.helper.validateForExternalTag(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	tagID := d.Id()
	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") {
		hasChanges = true
		tag := britive.Tag{}
		tag.Name = d.Get("name").(string)
		tag.Description = d.Get("description").(string)

		log.Printf("[INFO] Updating tag: %#v", tag)
		ut, err := c.UpdateTag(tagID, tag)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated tag: %#v", ut)
		d.SetId(ut.ID)
	}
	if d.HasChange("disabled") {
		hasChanges = true
		disabled := d.Get("disabled").(bool)

		log.Printf("[INFO] Updating status disabled: %t of tag: %s", disabled, tagID)
		ut, err := c.EnableOrDisableTag(tagID, disabled)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated status of tag: %#v", ut)
		d.SetId(ut.ID)
	}
	if d.HasChange("requestable") || d.HasChange("attributes") {
		hasChanges = true
		if diags := rt.helper.updateTagAttributes(c, tagID, d); diags != nil {
			return diags
		}
	}
	if hasChanges {
		return rt.resourceRead(ctx, d, m)
	}
	return nil
}

func (rt *ResourceTag) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	err := rt.helper.validateForExternalTag(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	tagID := d.Id()

	log.Printf("[INFO] Deleting tag: %s", tagID)
	err = c.DeleteTag(tagID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Tag %s deleted", tagID)
	d.SetId("")

	return diags
}

func (rt *ResourceTag) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rt.importHelper.ParseImportID([]string{"tags/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	tagName := d.Get("name").(string)

	if strings.TrimSpace(tagName) == "" {
		return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
	}

	log.Printf("[INFO] Importing tag: %s", tagName)

	tag, err := c.GetTagByName(tagName)
	if errors.Is(err, britive.ErrNotFound) {
		return nil, errs.NewNotFoundErrorf("tag %s", tagName)
	}
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Imported tag: %#v", tag)

	if tag.External.(bool) {
		return nil, fmt.Errorf("importing external tags is not supported. attempted to import tag '%s'", tagName)
	}

	d.SetId(tag.ID)

	err = rt.helper.mapModelToResource(tag, d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

//endregion

//ResourceTagHelper - Resource Tag helper functions
type ResourceTagHelper struct {
}

//NewResourceTagHelper - Initializes new tag resource helper
func NewResourceTagHelper() *ResourceTagHelper {
	return &ResourceTagHelper{}
}

//region Tag Resource helper functions

func (rth *ResourceTagHelper) mapModelToResource(tag *britive.Tag, d *schema.ResourceData, m interface{}) error {
	if err := d.Set("name", tag.Name); err != nil {
		return err
	}
	if err := d.Set("description", tag.Description); err != nil {
		return err
	}
	if len(tag.UserTagIdentityProviders) > 0 {
		if err := d.Set("identity_provider_id", tag.UserTagIdentityProviders[0].IdentityProvider.ID); err != nil {
			return err
		}
	}
	if err := d.Set("disabled", strings.EqualFold(tag.Status, "Inactive")); err != nil {
		return err
	}
	if err := d.Set("external", tag.External); err != nil {
		return err
	}
	if err := d.Set("requestable", tag.Requestable); err != nil {
		return err
	}
	attributes := make([]map[string]interface{}, len(tag.Attributes))
	for i, attr := range tag.Attributes {
		attributes[i] = map[string]interface{}{
			"attribute_name":  attr.AttributeName,
			"attribute_value": attr.AttributeValue,
		}
	}
	if err := d.Set("attributes", attributes); err != nil {
		return err
	}
	return nil
}

// hasAttributeConfig reports whether the user explicitly configured requestable
// or any attributes, meaning a PATCH to /user-tags is required.
func (rth *ResourceTagHelper) hasAttributeConfig(d *schema.ResourceData) bool {
	_, requestableSet := d.GetOkExists("requestable") //nolint:staticcheck
	return requestableSet || d.Get("attributes").(*schema.Set).Len() > 0
}

func (rth *ResourceTagHelper) updateTagAttributes(c *britive.Client, tagID string, d *schema.ResourceData) diag.Diagnostics {
	req := britive.TagAttributesUpdateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Attributes:  rth.buildTagAttributes(d),
	}

	// Only include requestable in the payload when the user explicitly configured it.
	// GetOkExists is designed for exactly this: TypeBool Optional+Computed with no Default.
	// It returns ok=false when the field is absent from config (r.Computed=true in the diff),
	// and ok=true when the user wrote requestable=true or requestable=false — correctly
	// handling the false case that HasChange misses during create (old=false, new=false).
	if v, ok := d.GetOkExists("requestable"); ok { //nolint:staticcheck
		b := v.(bool)
		req.Requestable = &b
	}

	log.Printf("[INFO] Updating tag attributes for tag %s: %#v", tagID, req)
	_, err := c.UpdateTagAttributes(tagID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func (rth *ResourceTagHelper) buildTagAttributes(d *schema.ResourceData) []britive.TagAttribute {
	attributesSet := d.Get("attributes").(*schema.Set)
	attributes := make([]britive.TagAttribute, 0, attributesSet.Len())
	for _, item := range attributesSet.List() {
		attr := item.(map[string]interface{})
		attributes = append(attributes, britive.TagAttribute{
			AttributeName:  attr["attribute_name"].(string),
			AttributeValue: attr["attribute_value"].(string),
		})
	}
	return attributes
}

func (rth *ResourceTagHelper) validateForExternalTag(d *schema.ResourceData, m interface{}) error {
	identityProviderID := d.Get("identity_provider_id").(string)
	if identityProviderID == "" {
		return nil
	}

	c := m.(*britive.Client)

	identityProvider, err := c.GetIdentityProvider(identityProviderID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(identityProvider.Type, "DEFAULT") {
		return fmt.Errorf("managing external tags is not supported. attempted to manage tag '%s'", d.Get("name").(string))
	}
	return nil
}

//endregion
