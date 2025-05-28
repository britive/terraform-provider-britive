package britive

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/britive/terraform-provider-britive/britive-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ResourceEntityGroup - Terraform Resource for Application Entity Group
type ResourceEntityGroup struct {
	Resource     *schema.Resource
	helper       *ResourceEntityGroupHelper
	importHelper *ImportHelper
}

// NewResourceEntityGroup - Initialization of new application entity group resource
func NewResourceEntityGroup(importHelper *ImportHelper) *ResourceEntityGroup {
	reg := &ResourceEntityGroup{
		helper:       NewResourceEntityGroupHelper(),
		importHelper: importHelper,
	}
	reg.Resource = &schema.Resource{
		CreateContext: reg.resourceCreate,
		ReadContext:   reg.resourceRead,
		UpdateContext: reg.resourceUpdate,
		DeleteContext: reg.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: reg.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The identity of the application entity of type environment group",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"application_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identity of the Britive application",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"entity_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the entity",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"entity_description": {
				Type:         schema.TypeString,
				Required:     true, //Should be set to optional when PAB-20647 is fixed.
				Description:  "The description of the entity",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"parent_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The parent id under which the environment group will be created",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
	return reg
}

//region Application Entity Group Resource Context Operations

func (reg *ResourceEntityGroup) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	applicationEntity := britive.ApplicationEntityGroup{}

	err := reg.helper.mapResourceToModel(d, m, &applicationEntity, false)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating new application entity group: %#v", applicationEntity)

	applicationID := d.Get("application_id").(string)

	ae, err := c.CreateEntityGroup(applicationEntity, applicationID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new application entity group: %#v", ae)
	d.SetId(reg.helper.generateUniqueID(applicationID, ae.EntityID))
	reg.resourceRead(ctx, d, m)
	return diags
}

func (reg *ResourceEntityGroup) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics

	err := reg.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (reg *ResourceEntityGroup) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var hasChanges bool
	if d.HasChange("application_id") || d.HasChange("entity_name") || d.HasChange("entity_description") || d.HasChange("parent_id") {
		hasChanges = true
		applicationID, _, err := reg.helper.parseUniqueID(d.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		applicationEntity := britive.ApplicationEntityGroup{}

		err = reg.helper.mapResourceToModel(d, m, &applicationEntity, true)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Updating the entity group %#v for application %s", applicationEntity, applicationID)

		ae, err := c.UpdateEntityGroup(applicationEntity, applicationID)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted updated application entity group: %#v", ae)
		d.SetId(reg.helper.generateUniqueID(applicationID, ae.EntityID))
	}
	if hasChanges {
		return reg.resourceRead(ctx, d, m)
	}
	return nil
}

func (reg *ResourceEntityGroup) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	applicationID, entityID, err := reg.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting entity group %s for application %s", entityID, applicationID)
	err = c.DeleteEntityGroup(applicationID, entityID)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted entity group %s for application %s", entityID, applicationID)
	d.SetId("")

	return diags
}

func (reg *ResourceEntityGroup) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	var err error
	if err := reg.importHelper.ParseImportID([]string{"apps/(?P<application_id>[^/]+)/root-environment-group/groups/(?P<entity_id>[^/]+)", "(?P<application_id>[^/]+)/groups/(?P<entity_id>[^/]+)"}, d); err != nil {
		return nil, err
	}

	applicationID := d.Get("application_id").(string)
	entityID := d.Get("entity_id").(string)
	if strings.TrimSpace(applicationID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("application_id")
	}
	if strings.TrimSpace(entityID) == "" {
		return nil, NewNotEmptyOrWhiteSpaceError("entity_id")
	}

	appEnvs := make([]britive.ApplicationEnvironment, 0)
	appEnvs, err = c.GetAppEnvs(applicationID, "environmentGroups")
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Importing entity group %s for application %s", entityID, applicationID)

	envIdList, err := c.GetEnvDetails(appEnvs, "id")
	if err != nil {
		return nil, err
	}

	for _, id := range envIdList {
		if id == entityID {
			d.SetId(reg.helper.generateUniqueID(applicationID, entityID))

			err = reg.helper.getAndMapModelToResource(d, m)
			if err != nil {
				return nil, err
			}
			log.Printf("[INFO] Imported entity group %s for application %s", entityID, applicationID)
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, NewNotFoundErrorf("entity group %s for application %s", entityID, applicationID)

}

//endregion

// ResourceEntityGroupHelper - Terraform Resource for Application Entity Group Helper
type ResourceEntityGroupHelper struct {
}

// NewResourceEntityGroupHelper - Initialization of new application entity group resource helper
func NewResourceEntityGroupHelper() *ResourceEntityGroupHelper {
	return &ResourceEntityGroupHelper{}
}

//region EntityGroup Resource helper functions

func (regh *ResourceEntityGroupHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, applicationEntity *britive.ApplicationEntityGroup, isUpdate bool) error {
	applicationEntity.Name = d.Get("entity_name").(string)
	applicationEntity.Description = d.Get("entity_description").(string)
	applicationEntity.ParentID = d.Get("parent_id").(string)
	applicationEntity.EntityID = d.Get("entity_id").(string)

	return nil
}

func (regh *ResourceEntityGroupHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	applicationID, entityID, err := regh.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading entity group %s for application %s", entityID, applicationID)

	appRootEnvironmentGroup, err := c.GetApplicationRootEnvironmentGroup(applicationID)
	if err != nil || appRootEnvironmentGroup == nil {
		return err
	}

	for _, association := range appRootEnvironmentGroup.EnvironmentGroups {
		if association.ID == entityID {
			log.Printf("[INFO] Received entity group: %#v", association)
			// To not allow the import of root environment group
			if association.ParentID == "" {
				return fmt.Errorf("`parent_id` cannot be empty")
			}
			if err := d.Set("entity_id", entityID); err != nil {
				return err
			}
			if err := d.Set("entity_name", association.Name); err != nil {
				return err
			}
			if err := d.Set("entity_description", association.Description); err != nil {
				return err
			}
			if err := d.Set("parent_id", association.ParentID); err != nil {
				return err
			}
			return nil
		}
	}

	return NewNotFoundErrorf("entity group %s for application %s", entityID, applicationID)
}

func (resourceEntityGroupHelper *ResourceEntityGroupHelper) generateUniqueID(applicationID, entityID string) string {
	return fmt.Sprintf("apps/%s/root-environment-group/groups/%s", applicationID, entityID)
}

func (resourceEntityGroupHelper *ResourceEntityGroupHelper) parseUniqueID(ID string) (applicationID, entityID string, err error) {
	applicationEntityParts := strings.Split(ID, "/")
	if len(applicationEntityParts) < 5 {
		err = NewInvalidResourceIDError("application entity group", ID)
		return
	}

	applicationID = applicationEntityParts[1]
	entityID = applicationEntityParts[4]
	return
}

//endregion
