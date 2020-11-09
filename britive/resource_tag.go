package britive

import (
	"context"
	"log"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//ResourceTag - Terraform Resource for Tag
type ResourceTag struct {
	Resource     *schema.Resource
	helper       *ResourceTagHelper
	importHelper *ImportHelper
}

//NewResourceTag - Initialises new tag resource
func NewResourceTag(importHelper *ImportHelper) *ResourceTag {
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
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the tag",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the tag",
			},
			"status": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The status of the tag",
				ValidateFunc: validation.StringInSlice([]string{"Active", "Inactive"}, false),
			},
			"user_tag_identity_providers": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "The identity provider list associated with tag",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"identity_provider": &schema.Schema{
							Type:        schema.TypeList,
							MaxItems:    1,
							Required:    true,
							Description: "The identity provider associated with tag",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "The id of the identity provider associated with tag",
									},
								},
							},
						},
					},
				},
			},
			"external": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "The flag whether tag is external or not",
			},
		},
	}
	return rt
}

//region Tag Resource Context Operations

func (rt *ResourceTag) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	tag := britive.Tag{}
	tag.Name = d.Get("name").(string)
	tag.Description = d.Get("description").(string)
	tag.Status = d.Get("status").(string)
	userTagIdentityProviders := d.Get("user_tag_identity_providers").([]interface{})

	for _, userTagIdentityProvider := range userTagIdentityProviders {
		utipm := userTagIdentityProvider.(map[string]interface{})

		ipm := utipm["identity_provider"].([]interface{})[0]
		ip := ipm.(map[string]interface{})

		utip := britive.UserTagIdentityProvider{
			IdentityProvider: britive.IdentityProvider{
				ID: ip["id"].(string),
			},
		}

		tag.UserTagIdentityProviders = append(tag.UserTagIdentityProviders, utip)
	}

	log.Printf("[INFO] Creating new tag: %#v", tag)
	ut, err := c.CreateTag(tag)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new tag: %#v", ut)
	d.SetId(ut.ID)

	return rt.resourceRead(ctx, d, m)
}

func (rt *ResourceTag) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID := d.Id()

	log.Printf("[INFO] Reading tag %s", tagID)
	tag, err := c.GetTag(tagID)
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
	if d.HasChange("status") {
		hasChanges = true
		status := d.Get("status").(string)

		log.Printf("[INFO] Updating status: %s of tag: %s", status, tagID)
		ut, err := c.EnableOrDisableTag(tagID, status)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated status of tag: %#v", ut)
		d.SetId(ut.ID)
	}
	if hasChanges {
		return rt.resourceRead(ctx, d, m)
	}
	return nil
}

func (rt *ResourceTag) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID := d.Id()

	log.Printf("[INFO] Deleting tag: %s", tagID)
	err := c.DeleteTag(tagID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Tag %s deleted", tagID)
	d.SetId("")

	return diags
}

func (rt *ResourceTag) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)

	if err := rt.importHelper.ParseImportID([]string{"user-tags/(?P<name>[^/]+)", "(?P<name>[^/]+)"}, d); err != nil {
		return nil, err
	}

	tagName := d.Get("name").(string)

	log.Printf("[INFO] Importing tag: %s", tagName)

	tag, err := c.GetTagByName(tagName)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Imported tag: %#v", tag)
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

//NewResourceTagHelper - Initialises new tag resource helper
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
	if err := d.Set("status", tag.Status); err != nil {
		return err
	}
	if err := d.Set("external", tag.External); err != nil {
		return err
	}
	utips := rth.mapIdentityProvidersModelToResource(&tag.UserTagIdentityProviders)
	if err := d.Set("user_tag_identity_providers", utips); err != nil {
		return err
	}
	return nil
}

func (rth *ResourceTagHelper) mapIdentityProvidersModelToResource(tagIdentityProviders *[]britive.UserTagIdentityProvider) []interface{} {
	if tagIdentityProviders != nil {
		utips := make([]interface{}, len(*tagIdentityProviders), len(*tagIdentityProviders))

		for i, tagIdentityProvider := range *tagIdentityProviders {
			utip := make(map[string]interface{})
			ip := make(map[string]interface{})
			ip["id"] = tagIdentityProvider.IdentityProvider.ID
			utip["identity_provider"] = []interface{}{ip}

			utips[i] = utip
		}
		return utips
	}
	return make([]interface{}, 0)
}

//endregion
