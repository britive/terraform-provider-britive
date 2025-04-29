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

// ResourceConstraint - Terraform Resource for Profile Permission Constraint
type ResourceConstraint struct {
	Resource     *schema.Resource
	helper       *ResourceConstraintHelper
	importHelper *imports.ImportHelper
}

// NewConstraint - Initialization of new permission constraint
func NewResourceConstraint(importHelper *imports.ImportHelper) *ResourceConstraint {
	rc := &ResourceConstraint{
		helper:       NewResourceConstraintHelper(),
		importHelper: importHelper,
	}
	rc.Resource = &schema.Resource{
		CreateContext: rc.resourceCreate,
		ReadContext:   rc.resourceRead,
		DeleteContext: rc.resourceDelete,
		Importer: &schema.ResourceImporter{
			State: rc.resourceStateImporter,
		},
		Schema: map[string]*schema.Schema{
			"profile_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The identifier of the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Name of the permission associated with the profile",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"permission_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "role",
				Description:  "The type of permission",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"constraint_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The constraint type for a given profile permission",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Name of the constraint",
			},
			"title": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Title of the condition constraint",
			},
			"expression": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Expression of the condition constraint",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the condition constraint",
			},
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			nameSet := d.Get("name").(string) != ""
			titleSet := d.Get("title").(string) != ""
			expressionSet := d.Get("expression").(string) != ""
			descriptionSet := d.Get("description").(string) != ""
			if nameSet && (titleSet || expressionSet || descriptionSet) {
				return fmt.Errorf("if `name` is set, then `title`, `expression`, and `description` cannot be set, and vice versa")
			}
			return nil
		},
	}
	return rc
}

//region Constraint Resource Context Operations

func (rc *ResourceConstraint) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)
	var diags diag.Diagnostics

	profileID := d.Get("profile_id").(string)
	permissionName := d.Get("permission_name").(string)
	permissionType := d.Get("permission_type").(string)
	constraintType := d.Get("constraint_type").(string)
	if strings.EqualFold(constraintType, "condition") {
		constraint := britive.ConditionConstraint{}
		err := rc.helper.mapConditionResourceToModel(d, m, &constraint)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Creating new condition constraint: %#v", constraint)

		co, err := c.CreateConditionConstraint(profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted new condition constraint: %#v", co)
		d.SetId(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, co.Title))
		rc.resourceRead(ctx, d, m)
	} else {
		constraint := britive.Constraint{}
		err := rc.helper.mapResourceToModel(d, m, &constraint)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Creating new constraint: %#v", constraint)
		co, err := c.CreateConstraint(profileID, permissionName, permissionType, constraintType, constraint)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("[INFO] Submitted new constraint: %#v", constraint)
		d.SetId(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, co.Name))
		rc.resourceRead(ctx, d, m)
	}

	return diags
}

func (rpc *ResourceConstraint) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	err := rpc.helper.getAndMapModelToResource(d, m)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func (rc *ResourceConstraint) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*britive.Client)

	var diags diag.Diagnostics

	profileID, permissionName, permissionType, constraintType, constraintName, err := rc.helper.parseUniqueID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting constraint: %s for permission %s of profile %s", constraintName, permissionName, profileID)
	err = c.DeleteConstraint(profileID, permissionName, permissionType, constraintType, constraintName)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleted constraint %s from profile %s for permission %s", constraintName, profileID, permissionName)
	d.SetId("")

	return diags
}

func (rc *ResourceConstraint) resourceStateImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*britive.Client)
	importConstraintType, err := rc.importHelper.FetchImportFieldValue([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)"}, d, "constraint_type")
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(importConstraintType, "condition") {
		if err := rc.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<title>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<title>[^/]+)"}, d); err != nil {
			return nil, err
		}

		profileID := d.Get("profile_id").(string)
		permissionName := d.Get("permission_name").(string)
		permissionType := d.Get("permission_type").(string)
		constraintType := d.Get("constraint_type").(string)
		constraintTitle := d.Get("title").(string)
		if strings.TrimSpace(profileID) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
		}
		if strings.TrimSpace(permissionName) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_name")
		}
		if strings.TrimSpace(permissionType) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_type")
		}
		if strings.TrimSpace(constraintType) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("constraint_type")
		}
		if strings.TrimSpace(constraintTitle) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("title")
		}

		log.Printf("[INFO] Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintTitle)
		constraintResult, err := c.GetConditionConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("Constraint Type %s for profile %s of permission %s", constraintType, profileID, permissionName)
		}
		if err != nil {
			return nil, err
		}

		if !britive.ConditionConstraintEqual(constraintTitle, constraintResult.Result[0].Expression, constraintResult.Result[0].Description, constraintResult) {
			return nil, errs.NewNotFoundErrorf("Constraint %s of type %s for profile %s of permission %s", constraintTitle, constraintType, profileID, permissionName)
		}

		d.SetId(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, constraintTitle))

		err = rc.helper.getAndMapModelToResource(d, m)
		if err != nil {
			return nil, err
		}
		log.Printf("[INFO] Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintTitle)
		return []*schema.ResourceData{d}, nil
	} else {
		if err := rc.importHelper.ParseImportID([]string{"paps/(?P<profile_id>[^/]+)/permissions/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/constraints/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)", "(?P<profile_id>[^/]+)/(?P<permission_name>[^/]+)/(?P<permission_type>[^/]+)/(?P<constraint_type>[^/]+)/(?P<name>[^/]+)"}, d); err != nil {
			return nil, err
		}

		profileID := d.Get("profile_id").(string)
		permissionName := d.Get("permission_name").(string)
		permissionType := d.Get("permission_type").(string)
		constraintType := d.Get("constraint_type").(string)
		constraintName := d.Get("name").(string)
		if strings.TrimSpace(profileID) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("profile_id")
		}
		if strings.TrimSpace(permissionName) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_name")
		}
		if strings.TrimSpace(permissionType) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("permission_type")
		}
		if strings.TrimSpace(constraintType) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("constraint_type")
		}
		if strings.TrimSpace(constraintName) == "" {
			return nil, errs.NewNotEmptyOrWhiteSpaceError("name")
		}

		log.Printf("[INFO] Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintName)
		constraintResult, err := c.GetConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			return nil, errs.NewNotFoundErrorf("Constraint Type %s for profile %s of permission %s", constraintType, profileID, permissionName)
		}
		if err != nil {
			return nil, err
		}

		if !britive.ConstraintEqual(constraintName, constraintResult) {
			return nil, errs.NewNotFoundErrorf("Constraint %s of type %s for profile %s of permission %s", constraintName, constraintType, profileID, permissionName)
		}

		d.SetId(rc.helper.generateUniqueID(profileID, permissionName, permissionType, constraintType, constraintName))

		err = rc.helper.getAndMapModelToResource(d, m)
		if err != nil {
			return nil, err
		}
		log.Printf("[INFO] Importing Constraint: %s/%s/%s/%s/%s", profileID, permissionName, permissionType, constraintType, constraintName)
		return []*schema.ResourceData{d}, nil
	}
}

//endregion

// ResourceConstraintHelper - Terraform Resource for Constraints Helper
type ResourceConstraintHelper struct {
}

// NewResourceConstraint - Initialization of new constraints resource helper
func NewResourceConstraintHelper() *ResourceConstraintHelper {
	return &ResourceConstraintHelper{}
}

//region Constraint Resource helper functions

func (rch *ResourceConstraintHelper) mapResourceToModel(d *schema.ResourceData, m interface{}, constraint *britive.Constraint) error {

	constraint.Name = d.Get("name").(string)

	return nil
}

func (rch *ResourceConstraintHelper) mapConditionResourceToModel(d *schema.ResourceData, m interface{}, constraint *britive.ConditionConstraint) error {

	constraint.Title = d.Get("title").(string)
	constraint.Expression = d.Get("expression").(string)
	constraint.Description = d.Get("description").(string)

	return nil
}

func (rch *ResourceConstraintHelper) getAndMapModelToResource(d *schema.ResourceData, m interface{}) error {
	c := m.(*britive.Client)

	profileID, permissionName, permissionType, constraintType, constraintName, err := rch.parseUniqueID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading constraint %s for the permission %s", constraintName, permissionName)

	if strings.EqualFold(constraintType, "condition") {
		constraintResult, err := c.GetConditionConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			return errs.NewNotFoundErrorf("Constraint %s in permission %s for profile id %s", constraintName, permissionName, profileID)
		}
		if err != nil {
			return err
		}

		newTitle := d.Get("title")
		newExpression := d.Get("expression")
		newDescription := d.Get("description")
		if britive.ConditionConstraintEqual(newTitle.(string), newExpression.(string), newDescription.(string), constraintResult) {
			if err := d.Set("title", newTitle); err != nil {
				return err
			}
			if err := d.Set("expression", newExpression); err != nil {
				return err
			}
			if err := d.Set("description", newDescription); err != nil {
				return err
			}
		} else {
			for _, rule := range constraintResult.Result {
				if err := d.Set("title", rule.Title); err != nil {
					return err
				}
				if err := d.Set("expression", rule.Expression); err != nil {
					return err
				}
				if err := d.Set("description", rule.Description); err != nil {
					return err
				}
			}
		}
	} else {
		constraintResult, err := c.GetConstraint(profileID, permissionName, permissionType, constraintType)
		if errors.Is(err, britive.ErrNotFound) {
			return errs.NewNotFoundErrorf("Constraint %s in permission %s for profile id %s", constraintName, permissionName, profileID)
		}
		if err != nil {
			return err
		}

		newName := d.Get("name")

		if britive.ConstraintEqual(newName.(string), constraintResult) {
			if err := d.Set("name", newName); err != nil {
				return err
			}
		} else {
			for _, rule := range constraintResult.Result {
				if err := d.Set("name", rule.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (resourceConstraintHelper *ResourceConstraintHelper) generateUniqueID(profileId, permissionName, permissionType, constraintType, constraintName string) string {
	return fmt.Sprintf("paps/%s/permissions/%s/%s/constraints/%s/%s", profileId, permissionName, permissionType, constraintType, constraintName)
}

func (resourceConstraintHelper *ResourceConstraintHelper) parseUniqueID(ID string) (profileId, permissionName, permissionType, constraintType, constraintName string, err error) {
	constraintParts := strings.Split(ID, "/")
	if len(constraintParts) < 8 {
		err = errs.NewInvalidResourceIDError("Constraint", ID)
		return
	}

	profileId = constraintParts[1]
	permissionName = constraintParts[3]
	permissionType = constraintParts[4]
	constraintType = constraintParts[6]
	constraintName = constraintParts[7]
	return
}

//endregion
