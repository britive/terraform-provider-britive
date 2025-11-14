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

// ResourceResponseTemplate - Terraform Resource for Response Template
type ResourceResponseTemplate struct {
	Resource     *schema.Resource
	helper       *ResourceResponseTemplateHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

// NewResourceResponseTemplate - Initialization of new response template resource
func NewResourceResponseTemplate(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceResponseTemplate {
	rrt := &ResourceResponseTemplate{
		helper:       NewResourceResponseTemplateHelper(),
		importHelper: importHelper,
		validation:   v,
	}
	rrt.Resource = &schema.Resource{
		CreateContext: rrt.resourceCreate,
		ReadContext:   rrt.resourceRead,
		UpdateContext: rrt.resourceUpdate,
		DeleteContext: rrt.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rrt.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the response template.",
				ValidateFunc: rrt.validation.StringWithNoSpecialChar,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the response template.",
			},
			"is_console_access_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Boolean flag to enable console access.",
			},
			"show_on_ui": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Boolean flag to determine if the template is visible on the UI.",
			},
			"template_data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The template content with placeholders.",
			},
			"template_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the response template.",
			},
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			isConsoleAccessEnabled := d.Get("is_console_access_enabled").(bool)
			showOnUI := d.Get("show_on_ui").(bool)

			if isConsoleAccessEnabled && showOnUI {
				return fmt.Errorf("both 'is_console_access_enabled' and 'show_on_ui' cannot be true at the same time")
			}

			return nil
		},
	}
	return rrt
}

func (rrt *ResourceResponseTemplate) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	template := &britive.ResponseTemplate{}
	rrt.helper.mapResourceToModel(d, m, template, false)

	log.Printf("[INFO] Creating response template: %#v", template)

	resp, err := c.CreateResponseTemplate(*template)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(rrt.helper.generateUniqueID(resp.TemplateID))
	log.Printf("[INFO] Created response template: %#v", template)
	return rrt.resourceRead(ctx, d, m)
}

func (rrt *ResourceResponseTemplate) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	templateID, err := rrt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(errs.NewInvalidResourceIDError("response template", d.Id()))
	}

	log.Printf("[INFO] Reading response template: %s", templateID)

	resp, err := c.GetResponseTemplate(templateID)
	if err != nil {
		if errors.Is(err, britive.ErrNotFound) {
			return diag.FromErr(errs.NewNotFoundErrorf("response template %s", templateID))
		}
		return diag.FromErr(err)
	}

	rrt.helper.getAndMapModelToResource(d, resp)
	log.Printf("[INFO] Received response template: %s", templateID)
	return diags
}

func (rrt *ResourceResponseTemplate) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	templateID, err := rrt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var hasChanges bool
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("is_console_access_enabled") || d.HasChange("show_on_ui") || d.HasChange("template_data") {
		hasChanges = true

		template := &britive.ResponseTemplate{}
		rrt.helper.mapResourceToModel(d, m, template, true)

		log.Printf("[INFO] Updating response template: %s", templateID)

		_, err := c.UpdateResponseTemplate(templateID, *template)
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[INFO] Updated response template: %s", templateID)
	}

	if hasChanges {
		return rrt.resourceRead(ctx, d, m)
	}

	return diags
}

func (rrt *ResourceResponseTemplate) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	var diags diag.Diagnostics

	templateID, err := rrt.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting response template: %s", templateID)

	err = c.DeleteResponseTemplate(templateID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Resource template %s deleted", templateID)
	d.SetId("")
	return diags
}

// ResourceResponseTemplateHelper - Helper for Response Template Resource
type ResourceResponseTemplateHelper struct{}

func NewResourceResponseTemplateHelper() *ResourceResponseTemplateHelper {
	return &ResourceResponseTemplateHelper{}
}

func (helper *ResourceResponseTemplateHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, template *britive.ResponseTemplate, isUpdate bool) {
	template.Name = d.Get("name").(string)
	template.Description = d.Get("description").(string)
	template.IsConsoleAccessEnabled = d.Get("is_console_access_enabled").(bool)
	if template.IsConsoleAccessEnabled {
		template.ShowOnUI = false
	} else {
		template.ShowOnUI = d.Get("show_on_ui").(bool)
	}
	template.TemplateData = d.Get("template_data").(string)
}

func (helper *ResourceResponseTemplateHelper) getAndMapModelToResource(d *schema.ResourceData, template *britive.ResponseTemplate) {
	d.Set("name", template.Name)
	d.Set("description", template.Description)
	d.Set("is_console_access_enabled", template.IsConsoleAccessEnabled)
	d.Set("show_on_ui", template.ShowOnUI)
	d.Set("template_data", template.TemplateData)
}

func (helper *ResourceResponseTemplateHelper) generateUniqueID(templateID string) string {
	return fmt.Sprintf("resource-manager/response-templates/%s", templateID)
}

func (helper *ResourceResponseTemplateHelper) parseUniqueID(ID string) (responseTemplateID string, err error) {
	responseTemplatesParts := strings.Split(ID, "/")

	if len(responseTemplatesParts) < 3 {
		err = errs.NewInvalidResourceIDError("responseTemplates", ID)
		return
	}

	responseTemplateID = responseTemplatesParts[2]
	return
}

func (rrt *ResourceResponseTemplate) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	providerMeta := m.(*britive.ProviderMeta)
	c := providerMeta.Client

	if err := rrt.importHelper.ParseImportID([]string{"resource-manager/response-templates/(?P<id>[^/]+)"}, d); err != nil {
		return nil, err
	}
	templateID := d.Id()
	log.Printf("[INFO] Importing response template: %s", templateID)

	resp, err := c.GetResponseTemplate(templateID)
	if err != nil {
		return nil, err
	}

	d.SetId(rrt.helper.generateUniqueID(resp.TemplateID))
	rrt.helper.getAndMapModelToResource(d, resp)
	log.Printf("[INFO] Imported response template: %s", templateID)
	return []*schema.ResourceData{d}, nil
}
