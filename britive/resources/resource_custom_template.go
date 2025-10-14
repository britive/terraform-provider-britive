package resources

import (
	"context"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/britive/terraform-provider-britive/britive/helpers/imports"
	"github.com/britive/terraform-provider-britive/britive/helpers/validate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ResourceCustomTemplate struct {
	Resource     *schema.Resource
	helper       *ResourceCustomTemplateHelper
	validation   *validate.Validation
	importHelper *imports.ImportHelper
}

type ResourceCustomTemplateHelper struct{}

func NewResourceCustomTemplateHelper() *ResourceCustomTemplateHelper {
	return &ResourceCustomTemplateHelper{}
}

func NewResourceCustomTemplate(v *validate.Validation, importHelper *imports.ImportHelper) *ResourceCustomTemplate {
	rct := &ResourceCustomTemplate{
		helper:       NewResourceCustomTemplateHelper(),
		validation:   v,
		importHelper: importHelper,
	}
	rct.Resource = &schema.Resource{
		CreateContext: rct.resourceCreate,
		ReadContext:   rct.resourceRead,
		UpdateContext: rct.resourceUpdate,
		DeleteContext: rct.resourceDelete,
		Schema: map[string]*schema.Schema{
			"template_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "cutom_app",
				Description: "Custom application name",
			},
			"template": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Custome template file",
			},
		},
	}

	return rct
}

func (rct *ResourceCustomTemplate) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	fileContent, fileName := rct.helper.mapResourceToModel(d)

	err := c.CreateCustomTemplate(fileContent, fileName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("apps")

	return nil
}

func (rct *ResourceCustomTemplate) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func (rct *ResourceCustomTemplate) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func (rct *ResourceCustomTemplate) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

func (helper *ResourceCustomTemplateHelper) mapResourceToModel(d *schema.ResourceData) (string, string) {
	fileContent := d.Get("file").(string)
	fileName, _ := d.GetOk("file_name")
	return fileContent, fileName.(string)
}
