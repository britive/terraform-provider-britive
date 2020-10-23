package britive

import (
	"context"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTag() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTagCreate,
		ReadContext:   resourceTagRead,
		UpdateContext: resourceTagUpdate,
		DeleteContext: resourceTagDelete,
		Importer: &schema.ResourceImporter{
			State: resourceTagStateImporter,
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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The status of the tag",
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
			"user_count": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Number of users associated with the tag",
			},
		},
	}
}

func resourceTagCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

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

	ut, err := c.CreateTag(tag)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(ut.ID)

	resourceTagRead(ctx, d, m)

	return diags
}

func resourceTagRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := getAndSetTagToState(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func getAndSetTagToState(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	tagID := d.Id()

	tag, err := c.GetTag(tagID)
	if err != nil {
		return err
	}
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
	if err := d.Set("user_count", tag.UserCount); err != nil {
		return err
	}
	utips := flattenTagIdentityProviders(&tag.UserTagIdentityProviders)
	if err := d.Set("user_tag_identity_providers", utips); err != nil {
		return err
	}
	return nil
}

func resourceTagUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("name") || d.HasChange("description") {
		c := m.(*britive.Client)
		tagID := d.Id()
		tag := britive.Tag{}
		tag.Name = d.Get("name").(string)
		tag.Description = d.Get("description").(string)
		_, err := c.UpdateTag(tagID, tag)
		if err != nil {
			return diag.FromErr(err)
		}
		return resourceTagRead(ctx, d, m)
	}
	return nil
}

func resourceTagDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	tagID := d.Id()

	err := c.DeleteTag(tagID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags
}

func resourceTagStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := parseImportID([]string{"user-tags/(?P<id>[^/]+)", "(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	err := getAndSetTagToState(d, m)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func flattenTagIdentityProviders(tagIdentityProviders *[]britive.UserTagIdentityProvider) []interface{} {
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
